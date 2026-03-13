package delivery

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

type SQLRepository struct {
	db *sqlx.DB
}

func NewSQLRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func NewRepository(db ...*sqlx.DB) Repository {
	if len(db) > 0 && db[0] != nil {
		return NewSQLRepository(db[0])
	}
	return newMemoryRepository()
}

type recordRow struct {
	ID                int64      `db:"id"`
	UID               string     `db:"uid"`
	RequestID         int64      `db:"request_id"`
	RequestUID        string     `db:"request_uid"`
	CorrelationID     string     `db:"correlation_id"`
	ServerID          int64      `db:"server_id"`
	ServerName        string     `db:"server_name"`
	ServerCode        string     `db:"server_code"`
	SystemType        string     `db:"system_type"`
	AttemptNumber     int        `db:"attempt_number"`
	Status            string     `db:"status"`
	HTTPStatus        *int       `db:"http_status"`
	ResponseBody      string     `db:"response_body"`
	ResponseContentType string   `db:"response_content_type"`
	ResponseBodyFiltered bool    `db:"response_body_filtered"`
	ResponseSummary   json.RawMessage `db:"response_summary"`
	ErrorMessage      string     `db:"error_message"`
	SubmissionHoldReason string  `db:"submission_hold_reason"`
	NextEligibleAt    *time.Time `db:"next_eligible_at"`
	HoldPolicySource  string     `db:"hold_policy_source"`
	TerminalReason    string     `db:"terminal_reason"`
	AsyncTaskID       *int64     `db:"async_task_id"`
	AsyncTaskUID      string     `db:"async_task_uid"`
	AsyncCurrentState string     `db:"async_current_state"`
	AsyncRemoteJobID  string     `db:"async_remote_job_id"`
	AsyncPollURL      string     `db:"async_poll_url"`
	StartedAt         *time.Time `db:"started_at"`
	FinishedAt        *time.Time `db:"finished_at"`
	RetryAt           *time.Time `db:"retry_at"`
	CreatedAt         time.Time  `db:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at"`
}

func normalizeListQuery(query ListQuery) ListQuery {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 25
	}

	sortField := strings.TrimSpace(query.SortField)
	switch sortField {
	case "uid", "requestUid", "serverName", "attemptNumber", "status", "startedAt", "finishedAt", "retryAt", "updatedAt":
	default:
		sortField = "createdAt"
	}

	sortOrder := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if sortOrder != "asc" {
		sortOrder = "desc"
	}

	return ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Filter:    strings.TrimSpace(query.Filter),
		Status:    strings.ToLower(strings.TrimSpace(query.Status)),
		Server:    strings.TrimSpace(query.Server),
		Date:      query.Date,
	}
}

func (r *SQLRepository) ListDeliveries(ctx context.Context, query ListQuery) (ListResult, error) {
	q := normalizeListQuery(query)
	offset := (q.Page - 1) * q.PageSize

	conditions := make([]string, 0, 4)
	args := make([]any, 0, 8)
	if q.Filter != "" {
		args = append(args, "%"+q.Filter+"%")
		needle := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, `(
			d.uid::text ILIKE `+needle+` OR
			COALESCE(r.uid::text, '') ILIKE `+needle+` OR
			COALESCE(s.name, '') ILIKE `+needle+` OR
			COALESCE(s.code, '') ILIKE `+needle+` OR
			COALESCE(d.error_message, '') ILIKE `+needle+`
		)`)
	}
	if q.Status != "" {
		args = append(args, q.Status)
		conditions = append(conditions, fmt.Sprintf("d.status = $%d", len(args)))
	}
	if q.Server != "" {
		if id, err := strconv.ParseInt(q.Server, 10, 64); err == nil && id > 0 {
			args = append(args, id)
			conditions = append(conditions, fmt.Sprintf("d.server_id = $%d", len(args)))
		} else {
			args = append(args, "%"+q.Server+"%")
			needle := fmt.Sprintf("$%d", len(args))
			conditions = append(conditions, `(COALESCE(s.name, '') ILIKE `+needle+` OR COALESCE(s.code, '') ILIKE `+needle+`)`)
		}
	}
	if q.Date != nil {
		args = append(args, q.Date.UTC().Format("2006-01-02"))
		conditions = append(conditions, fmt.Sprintf("DATE(d.created_at) = $%d::date", len(args)))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	baseFrom := `
		FROM delivery_attempts d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN integration_servers s ON s.id = d.server_id
		LEFT JOIN async_tasks a ON a.delivery_attempt_id = d.id
	`

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) `+baseFrom+whereClause, args...); err != nil {
		return ListResult{}, fmt.Errorf("count delivery attempts: %w", err)
	}

	selectArgs := append([]any{}, args...)
	selectArgs = append(selectArgs, q.PageSize, offset)
	querySQL := `
		SELECT d.id, d.uid::text AS uid, d.request_id, COALESCE(r.uid::text, '') AS request_uid, COALESCE(r.correlation_id, '') AS correlation_id,
		       d.server_id, COALESCE(s.name, '') AS server_name, COALESCE(s.code, '') AS server_code, COALESCE(s.system_type, '') AS system_type,
		       d.attempt_number, d.status, d.http_status, COALESCE(d.response_body, '') AS response_body,
		       COALESCE(d.response_content_type, '') AS response_content_type, d.response_body_filtered, d.response_summary,
		       COALESCE(d.error_message, '') AS error_message,
		       COALESCE(d.submission_hold_reason, '') AS submission_hold_reason, d.next_eligible_at, COALESCE(d.hold_policy_source, '') AS hold_policy_source,
		       COALESCE(d.terminal_reason, '') AS terminal_reason,
		       a.id AS async_task_id, COALESCE(a.uid::text, '') AS async_task_uid,
		       COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN '' ELSE a.remote_status END) AS async_current_state,
		       COALESCE(a.remote_job_id, '') AS async_remote_job_id, COALESCE(a.poll_url, '') AS async_poll_url,
		       d.started_at, d.finished_at, d.retry_at, d.created_at, d.updated_at
	` + baseFrom + whereClause + fmt.Sprintf(
		" ORDER BY %s %s LIMIT $%d OFFSET $%d",
		resolveSortColumn(q.SortField),
		strings.ToUpper(q.SortOrder),
		len(selectArgs)-1,
		len(selectArgs),
	)

	rows := []recordRow{}
	if err := r.db.SelectContext(ctx, &rows, querySQL, selectArgs...); err != nil {
		return ListResult{}, fmt.Errorf("list delivery attempts: %w", err)
	}

	items := make([]Record, 0, len(rows))
	for _, row := range rows {
		items = append(items, decodeRow(row))
	}

	return ListResult{
		Items:    items,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *SQLRepository) GetDeliveryByID(ctx context.Context, id int64) (Record, error) {
	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT d.id, d.uid::text AS uid, d.request_id, COALESCE(r.uid::text, '') AS request_uid, COALESCE(r.correlation_id, '') AS correlation_id,
		       d.server_id, COALESCE(s.name, '') AS server_name, COALESCE(s.code, '') AS server_code, COALESCE(s.system_type, '') AS system_type,
		       d.attempt_number, d.status, d.http_status, COALESCE(d.response_body, '') AS response_body,
		       COALESCE(d.response_content_type, '') AS response_content_type, d.response_body_filtered, d.response_summary,
		       COALESCE(d.error_message, '') AS error_message,
		       COALESCE(d.submission_hold_reason, '') AS submission_hold_reason, d.next_eligible_at, COALESCE(d.hold_policy_source, '') AS hold_policy_source,
		       COALESCE(d.terminal_reason, '') AS terminal_reason,
		       a.id AS async_task_id, COALESCE(a.uid::text, '') AS async_task_uid,
		       COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN '' ELSE a.remote_status END) AS async_current_state,
		       COALESCE(a.remote_job_id, '') AS async_remote_job_id, COALESCE(a.poll_url, '') AS async_poll_url,
		       d.started_at, d.finished_at, d.retry_at, d.created_at, d.updated_at
		FROM delivery_attempts d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN integration_servers s ON s.id = d.server_id
		LEFT JOIN async_tasks a ON a.delivery_attempt_id = d.id
		WHERE d.id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("get delivery attempt: %w", err)
	}
	return decodeRow(row), nil
}

func (r *SQLRepository) CreateDelivery(ctx context.Context, params CreateParams) (Record, error) {
	responseSummary, err := json.Marshal(cloneJSONMap(params.ResponseSummary))
	if err != nil {
		return Record{}, fmt.Errorf("marshal delivery response summary: %w", err)
	}
	var id int64
	if err := r.db.GetContext(ctx, &id, `
		INSERT INTO delivery_attempts (
			uid, request_id, server_id, attempt_number, status, http_status, response_body, response_content_type,
			response_body_filtered, response_summary, error_message, submission_hold_reason, next_eligible_at, hold_policy_source,
			terminal_reason, started_at, finished_at, retry_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11, $12, $13, $14, $15, $16, $17, $18, NOW(), NOW())
		RETURNING id
	`,
		params.UID,
		params.RequestID,
		params.ServerID,
		params.AttemptNumber,
		params.Status,
		params.HTTPStatus,
		params.ResponseBody,
		params.ResponseContentType,
		params.ResponseBodyFiltered,
		string(responseSummary),
		params.ErrorMessage,
		params.SubmissionHoldReason,
		params.NextEligibleAt,
		params.HoldPolicySource,
		params.TerminalReason,
		params.StartedAt,
		params.FinishedAt,
		params.RetryAt,
	); err != nil {
		return Record{}, fmt.Errorf("create delivery attempt: %w", err)
	}

	return r.GetDeliveryByID(ctx, id)
}

func (r *SQLRepository) UpdateDelivery(ctx context.Context, params UpdateParams) (Record, error) {
	responseSummary, err := json.Marshal(cloneJSONMap(params.ResponseSummary))
	if err != nil {
		return Record{}, fmt.Errorf("marshal delivery response summary: %w", err)
	}
	var id int64
	if err := r.db.GetContext(ctx, &id, `
		UPDATE delivery_attempts
		SET status = $2,
		    http_status = $3,
		    response_body = $4,
		    response_content_type = $5,
		    response_body_filtered = $6,
		    response_summary = $7::jsonb,
		    error_message = $8,
		    submission_hold_reason = $9,
		    next_eligible_at = $10,
		    hold_policy_source = $11,
		    terminal_reason = $12,
		    started_at = $13,
		    finished_at = $14,
		    retry_at = $15,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`,
		params.ID,
		params.Status,
		params.HTTPStatus,
		params.ResponseBody,
		params.ResponseContentType,
		params.ResponseBodyFiltered,
		string(responseSummary),
		params.ErrorMessage,
		params.SubmissionHoldReason,
		params.NextEligibleAt,
		params.HoldPolicySource,
		params.TerminalReason,
		params.StartedAt,
		params.FinishedAt,
		params.RetryAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("update delivery attempt: %w", err)
	}

	return r.GetDeliveryByID(ctx, id)
}

func resolveSortColumn(sortField string) string {
	switch sortField {
	case "uid":
		return "d.uid"
	case "requestUid":
		return "r.uid"
	case "serverName":
		return "s.name"
	case "attemptNumber":
		return "d.attempt_number"
	case "status":
		return "d.status"
	case "startedAt":
		return "d.started_at"
	case "finishedAt":
		return "d.finished_at"
	case "retryAt":
		return "d.retry_at"
	case "updatedAt":
		return "d.updated_at"
	default:
		return "d.created_at"
	}
}

func decodeRow(row recordRow) Record {
	responseSummary := map[string]any{}
	if len(row.ResponseSummary) > 0 {
		_ = json.Unmarshal(row.ResponseSummary, &responseSummary)
	}
	return Record{
		ID:                row.ID,
		UID:               row.UID,
		RequestID:         row.RequestID,
		RequestUID:        row.RequestUID,
		CorrelationID:     row.CorrelationID,
		ServerID:          row.ServerID,
		ServerName:        row.ServerName,
		ServerCode:        row.ServerCode,
		SystemType:        row.SystemType,
		AttemptNumber:     row.AttemptNumber,
		Status:            row.Status,
		HTTPStatus:        cloneIntPtr(row.HTTPStatus),
		ResponseBody:      row.ResponseBody,
		ResponseContentType: row.ResponseContentType,
		ResponseBodyFiltered: row.ResponseBodyFiltered,
		ResponseSummary:   responseSummary,
		ErrorMessage:      row.ErrorMessage,
		SubmissionHoldReason: row.SubmissionHoldReason,
		NextEligibleAt:    cloneTimePtr(row.NextEligibleAt),
		HoldPolicySource:  row.HoldPolicySource,
		TerminalReason:    row.TerminalReason,
		SubmissionMode:    submissionMode(row.AsyncTaskID),
		AsyncTaskID:       cloneInt64Ptr(row.AsyncTaskID),
		AsyncTaskUID:      row.AsyncTaskUID,
		AsyncCurrentState: row.AsyncCurrentState,
		AsyncRemoteJobID:  row.AsyncRemoteJobID,
		AsyncPollURL:      row.AsyncPollURL,
		AwaitingAsync:     row.AsyncTaskID != nil && row.AsyncCurrentState != "" && row.AsyncCurrentState != StatusSucceeded && row.AsyncCurrentState != StatusFailed,
		StartedAt:         cloneTimePtr(row.StartedAt),
		FinishedAt:        cloneTimePtr(row.FinishedAt),
		RetryAt:           cloneTimePtr(row.RetryAt),
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}
}

type memoryRepository struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]Record
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		nextID: 1,
		items:  map[int64]Record{},
	}
}

func (r *memoryRepository) ListDeliveries(_ context.Context, query ListQuery) (ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := normalizeListQuery(query)
	filter := strings.ToLower(q.Filter)
	serverFilter := strings.ToLower(q.Server)
	dateValue := ""
	if q.Date != nil {
		dateValue = q.Date.UTC().Format("2006-01-02")
	}

	items := make([]Record, 0, len(r.items))
	for _, item := range r.items {
		if q.Status != "" && item.Status != q.Status {
			continue
		}
		if serverFilter != "" {
			if id, err := strconv.ParseInt(serverFilter, 10, 64); err == nil {
				if item.ServerID != id {
					continue
				}
			} else {
				searchableServer := strings.ToLower(item.ServerName)
				if !strings.Contains(searchableServer, serverFilter) {
					continue
				}
			}
		}
		if dateValue != "" && item.CreatedAt.UTC().Format("2006-01-02") != dateValue {
			continue
		}
		if filter != "" {
			searchable := strings.ToLower(strings.Join([]string{
				item.UID,
				item.RequestUID,
				item.ServerName,
				item.ErrorMessage,
			}, " "))
			if !strings.Contains(searchable, filter) {
				continue
			}
		}
		items = append(items, cloneRecord(item))
	}

	slices.SortFunc(items, func(a, b Record) int {
		var cmp int
		switch q.SortField {
		case "uid":
			cmp = strings.Compare(a.UID, b.UID)
		case "requestUid":
			cmp = strings.Compare(a.RequestUID, b.RequestUID)
		case "serverName":
			cmp = strings.Compare(a.ServerName, b.ServerName)
		case "attemptNumber":
			cmp = a.AttemptNumber - b.AttemptNumber
		case "status":
			cmp = strings.Compare(a.Status, b.Status)
		case "startedAt":
			cmp = compareOptionalTimes(a.StartedAt, b.StartedAt)
		case "finishedAt":
			cmp = compareOptionalTimes(a.FinishedAt, b.FinishedAt)
		case "retryAt":
			cmp = compareOptionalTimes(a.RetryAt, b.RetryAt)
		case "updatedAt":
			cmp = compareTimes(a.UpdatedAt, b.UpdatedAt)
		default:
			cmp = compareTimes(a.CreatedAt, b.CreatedAt)
		}
		if q.SortOrder == "desc" {
			return -cmp
		}
		return cmp
	})

	total := len(items)
	start := (q.Page - 1) * q.PageSize
	if start > total {
		start = total
	}
	end := start + q.PageSize
	if end > total {
		end = total
	}

	return ListResult{
		Items:    items[start:end],
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *memoryRepository) GetDeliveryByID(_ context.Context, id int64) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	return cloneRecord(item), nil
}

func (r *memoryRepository) CreateDelivery(_ context.Context, params CreateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextID
	r.nextID++
	now := time.Now().UTC()
	record := Record{
		ID:                   id,
		UID:                  params.UID,
		RequestID:            params.RequestID,
		RequestUID:           fmt.Sprintf("request-%d", params.RequestID),
		ServerID:             params.ServerID,
		ServerName:           fmt.Sprintf("Server #%d", params.ServerID),
		ServerCode:           fmt.Sprintf("server-%d", params.ServerID),
		AttemptNumber:        params.AttemptNumber,
		Status:               params.Status,
		HTTPStatus:           cloneIntPtr(params.HTTPStatus),
		ResponseBody:         params.ResponseBody,
		ResponseContentType:  params.ResponseContentType,
		ResponseBodyFiltered: params.ResponseBodyFiltered,
		ResponseSummary:      cloneJSONMap(params.ResponseSummary),
		ErrorMessage:         params.ErrorMessage,
		SubmissionHoldReason: params.SubmissionHoldReason,
		NextEligibleAt:       cloneTimePtr(params.NextEligibleAt),
		HoldPolicySource:     params.HoldPolicySource,
		TerminalReason:       params.TerminalReason,
		SubmissionMode:       "synchronous",
		StartedAt:            cloneTimePtr(params.StartedAt),
		FinishedAt:           cloneTimePtr(params.FinishedAt),
		RetryAt:              cloneTimePtr(params.RetryAt),
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	r.items[id] = record
	return cloneRecord(record), nil
}

func (r *memoryRepository) UpdateDelivery(_ context.Context, params UpdateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[params.ID]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	item.Status = params.Status
	item.HTTPStatus = cloneIntPtr(params.HTTPStatus)
	item.ResponseBody = params.ResponseBody
	item.ResponseContentType = params.ResponseContentType
	item.ResponseBodyFiltered = params.ResponseBodyFiltered
	item.ResponseSummary = cloneJSONMap(params.ResponseSummary)
	item.ErrorMessage = params.ErrorMessage
	item.SubmissionHoldReason = params.SubmissionHoldReason
	item.NextEligibleAt = cloneTimePtr(params.NextEligibleAt)
	item.HoldPolicySource = params.HoldPolicySource
	item.TerminalReason = params.TerminalReason
	item.StartedAt = cloneTimePtr(params.StartedAt)
	item.FinishedAt = cloneTimePtr(params.FinishedAt)
	item.RetryAt = cloneTimePtr(params.RetryAt)
	item.UpdatedAt = time.Now().UTC()
	r.items[params.ID] = item
	return cloneRecord(item), nil
}

func compareTimes(a time.Time, b time.Time) int {
	if a.Before(b) {
		return -1
	}
	if a.After(b) {
		return 1
	}
	return 0
}

func compareOptionalTimes(a *time.Time, b *time.Time) int {
	switch {
	case a == nil && b == nil:
		return 0
	case a == nil:
		return -1
	case b == nil:
		return 1
	default:
		return compareTimes(*a, *b)
	}
}

func cloneRecord(input Record) Record {
	input.HTTPStatus = cloneIntPtr(input.HTTPStatus)
	input.AsyncTaskID = cloneInt64Ptr(input.AsyncTaskID)
	input.ResponseSummary = cloneJSONMap(input.ResponseSummary)
	input.NextEligibleAt = cloneTimePtr(input.NextEligibleAt)
	input.StartedAt = cloneTimePtr(input.StartedAt)
	input.FinishedAt = cloneTimePtr(input.FinishedAt)
	input.RetryAt = cloneTimePtr(input.RetryAt)
	return input
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func submissionMode(asyncTaskID *int64) string {
	if asyncTaskID != nil {
		return "async"
	}
	return "synchronous"
}

func newUID() string {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		panic(err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return hex.EncodeToString(raw[0:4]) + "-" +
		hex.EncodeToString(raw[4:6]) + "-" +
		hex.EncodeToString(raw[6:8]) + "-" +
		hex.EncodeToString(raw[8:10]) + "-" +
		hex.EncodeToString(raw[10:16])
}
