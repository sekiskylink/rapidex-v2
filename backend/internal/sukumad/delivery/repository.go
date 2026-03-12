package delivery

import (
	"context"
	"crypto/rand"
	"database/sql"
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
	ID            int64      `db:"id"`
	UID           string     `db:"uid"`
	RequestID     int64      `db:"request_id"`
	RequestUID    string     `db:"request_uid"`
	ServerID      int64      `db:"server_id"`
	ServerName    string     `db:"server_name"`
	AttemptNumber int        `db:"attempt_number"`
	Status        string     `db:"status"`
	HTTPStatus    *int       `db:"http_status"`
	ResponseBody  string     `db:"response_body"`
	ErrorMessage  string     `db:"error_message"`
	StartedAt     *time.Time `db:"started_at"`
	FinishedAt    *time.Time `db:"finished_at"`
	RetryAt       *time.Time `db:"retry_at"`
	CreatedAt     time.Time  `db:"created_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
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
			d.uid ILIKE `+needle+` OR
			COALESCE(r.uid, '') ILIKE `+needle+` OR
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
	`

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) `+baseFrom+whereClause, args...); err != nil {
		return ListResult{}, fmt.Errorf("count delivery attempts: %w", err)
	}

	selectArgs := append([]any{}, args...)
	selectArgs = append(selectArgs, q.PageSize, offset)
	querySQL := `
		SELECT d.id, d.uid, d.request_id, COALESCE(r.uid, '') AS request_uid,
		       d.server_id, COALESCE(s.name, '') AS server_name,
		       d.attempt_number, d.status, d.http_status, d.response_body, d.error_message,
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
		SELECT d.id, d.uid, d.request_id, COALESCE(r.uid, '') AS request_uid,
		       d.server_id, COALESCE(s.name, '') AS server_name,
		       d.attempt_number, d.status, d.http_status, d.response_body, d.error_message,
		       d.started_at, d.finished_at, d.retry_at, d.created_at, d.updated_at
		FROM delivery_attempts d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN integration_servers s ON s.id = d.server_id
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
	var id int64
	if err := r.db.GetContext(ctx, &id, `
		INSERT INTO delivery_attempts (
			uid, request_id, server_id, attempt_number, status, http_status, response_body, error_message,
			started_at, finished_at, retry_at, created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id
	`,
		params.UID,
		params.RequestID,
		params.ServerID,
		params.AttemptNumber,
		params.Status,
		params.HTTPStatus,
		params.ResponseBody,
		params.ErrorMessage,
		params.StartedAt,
		params.FinishedAt,
		params.RetryAt,
	); err != nil {
		return Record{}, fmt.Errorf("create delivery attempt: %w", err)
	}

	return r.GetDeliveryByID(ctx, id)
}

func (r *SQLRepository) UpdateDelivery(ctx context.Context, params UpdateParams) (Record, error) {
	var id int64
	if err := r.db.GetContext(ctx, &id, `
		UPDATE delivery_attempts
		SET status = $2,
		    http_status = $3,
		    response_body = $4,
		    error_message = $5,
		    started_at = $6,
		    finished_at = $7,
		    retry_at = $8,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`,
		params.ID,
		params.Status,
		params.HTTPStatus,
		params.ResponseBody,
		params.ErrorMessage,
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
	return Record{
		ID:            row.ID,
		UID:           row.UID,
		RequestID:     row.RequestID,
		RequestUID:    row.RequestUID,
		ServerID:      row.ServerID,
		ServerName:    row.ServerName,
		AttemptNumber: row.AttemptNumber,
		Status:        row.Status,
		HTTPStatus:    cloneIntPtr(row.HTTPStatus),
		ResponseBody:  row.ResponseBody,
		ErrorMessage:  row.ErrorMessage,
		StartedAt:     cloneTimePtr(row.StartedAt),
		FinishedAt:    cloneTimePtr(row.FinishedAt),
		RetryAt:       cloneTimePtr(row.RetryAt),
		CreatedAt:     row.CreatedAt,
		UpdatedAt:     row.UpdatedAt,
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
		ID:            id,
		UID:           params.UID,
		RequestID:     params.RequestID,
		RequestUID:    fmt.Sprintf("request-%d", params.RequestID),
		ServerID:      params.ServerID,
		ServerName:    fmt.Sprintf("Server #%d", params.ServerID),
		AttemptNumber: params.AttemptNumber,
		Status:        params.Status,
		HTTPStatus:    cloneIntPtr(params.HTTPStatus),
		ResponseBody:  params.ResponseBody,
		ErrorMessage:  params.ErrorMessage,
		StartedAt:     cloneTimePtr(params.StartedAt),
		FinishedAt:    cloneTimePtr(params.FinishedAt),
		RetryAt:       cloneTimePtr(params.RetryAt),
		CreatedAt:     now,
		UpdatedAt:     now,
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
	item.ErrorMessage = params.ErrorMessage
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
