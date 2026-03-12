package async

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
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
	ID                int64           `db:"id"`
	UID               string          `db:"uid"`
	DeliveryAttemptID int64           `db:"delivery_attempt_id"`
	DeliveryUID       string          `db:"delivery_uid"`
	RequestID         int64           `db:"request_id"`
	RequestUID        string          `db:"request_uid"`
	RemoteJobID       string          `db:"remote_job_id"`
	PollURL           string          `db:"poll_url"`
	RemoteStatus      string          `db:"remote_status"`
	TerminalState     string          `db:"terminal_state"`
	NextPollAt        *time.Time      `db:"next_poll_at"`
	CompletedAt       *time.Time      `db:"completed_at"`
	RemoteResponse    json.RawMessage `db:"remote_response"`
	CreatedAt         time.Time       `db:"created_at"`
	UpdatedAt         time.Time       `db:"updated_at"`
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
	case "uid", "deliveryUid", "requestUid", "remoteJobId", "remoteStatus", "terminalState", "nextPollAt", "completedAt", "updatedAt":
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
	}
}

func (r *SQLRepository) ListTasks(ctx context.Context, query ListQuery) (ListResult, error) {
	q := normalizeListQuery(query)
	offset := (q.Page - 1) * q.PageSize

	conditions := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if q.Filter != "" {
		args = append(args, "%"+q.Filter+"%")
		needle := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, `(
			a.uid::text ILIKE `+needle+` OR
			COALESCE(d.uid::text, '') ILIKE `+needle+` OR
			COALESCE(rq.uid::text, '') ILIKE `+needle+` OR
			COALESCE(a.remote_job_id, '') ILIKE `+needle+` OR
			COALESCE(a.remote_status, '') ILIKE `+needle+` OR
			COALESCE(a.terminal_state, '') ILIKE `+needle+`
		)`)
	}
	if q.Status != "" {
		args = append(args, q.Status)
		bind := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, `(COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN 'pending' ELSE a.remote_status END) = `+bind+`)`)
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	baseFrom := `
		FROM async_tasks a
		LEFT JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
		LEFT JOIN exchange_requests rq ON rq.id = d.request_id
	`

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) `+baseFrom+whereClause, args...); err != nil {
		return ListResult{}, fmt.Errorf("count async tasks: %w", err)
	}

	selectArgs := append([]any{}, args...)
	selectArgs = append(selectArgs, q.PageSize, offset)
	querySQL := `
		SELECT a.id, a.uid::text AS uid, a.delivery_attempt_id, COALESCE(d.uid::text, '') AS delivery_uid,
		       COALESCE(d.request_id, 0) AS request_id, COALESCE(rq.uid::text, '') AS request_uid,
		       COALESCE(a.remote_job_id, '') AS remote_job_id, COALESCE(a.poll_url, '') AS poll_url,
		       COALESCE(a.remote_status, '') AS remote_status, COALESCE(a.terminal_state, '') AS terminal_state,
		       a.next_poll_at, a.completed_at, a.remote_response, a.created_at, a.updated_at
	` + baseFrom + whereClause + fmt.Sprintf(
		" ORDER BY %s %s LIMIT $%d OFFSET $%d",
		resolveSortColumn(q.SortField),
		strings.ToUpper(q.SortOrder),
		len(selectArgs)-1,
		len(selectArgs),
	)

	rows := []recordRow{}
	if err := r.db.SelectContext(ctx, &rows, querySQL, selectArgs...); err != nil {
		return ListResult{}, fmt.Errorf("list async tasks: %w", err)
	}

	items, err := decodeRows(rows)
	if err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Items:    items,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *SQLRepository) GetTaskByID(ctx context.Context, id int64) (Record, error) {
	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT a.id, a.uid::text AS uid, a.delivery_attempt_id, COALESCE(d.uid::text, '') AS delivery_uid,
		       COALESCE(d.request_id, 0) AS request_id, COALESCE(rq.uid::text, '') AS request_uid,
		       COALESCE(a.remote_job_id, '') AS remote_job_id, COALESCE(a.poll_url, '') AS poll_url,
		       COALESCE(a.remote_status, '') AS remote_status, COALESCE(a.terminal_state, '') AS terminal_state,
		       a.next_poll_at, a.completed_at, a.remote_response, a.created_at, a.updated_at
		FROM async_tasks a
		LEFT JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
		LEFT JOIN exchange_requests rq ON rq.id = d.request_id
		WHERE a.id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("get async task: %w", err)
	}
	return decodeRow(row)
}

func (r *SQLRepository) CreateTask(ctx context.Context, params CreateParams) (Record, error) {
	response, err := json.Marshal(cloneJSONMap(params.RemoteResponse))
	if err != nil {
		return Record{}, fmt.Errorf("marshal async remote response: %w", err)
	}

	var id int64
	if err := r.db.GetContext(ctx, &id, `
		INSERT INTO async_tasks (
			uid, delivery_attempt_id, remote_job_id, poll_url, remote_status, terminal_state,
			next_poll_at, completed_at, remote_response, created_at, updated_at
		)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), $7, $8, $9::jsonb, NOW(), NOW())
		RETURNING id
	`,
		params.UID,
		params.DeliveryAttemptID,
		params.RemoteJobID,
		params.PollURL,
		params.RemoteStatus,
		params.TerminalState,
		params.NextPollAt,
		params.CompletedAt,
		string(response),
	); err != nil {
		return Record{}, fmt.Errorf("create async task: %w", err)
	}

	return r.GetTaskByID(ctx, id)
}

func (r *SQLRepository) UpdateTask(ctx context.Context, params UpdateParams) (Record, error) {
	response, err := json.Marshal(cloneJSONMap(params.RemoteResponse))
	if err != nil {
		return Record{}, fmt.Errorf("marshal async remote response: %w", err)
	}

	var id int64
	if err := r.db.GetContext(ctx, &id, `
		UPDATE async_tasks
		SET remote_job_id = NULLIF($2, ''),
		    poll_url = NULLIF($3, ''),
		    remote_status = NULLIF($4, ''),
		    terminal_state = NULLIF($5, ''),
		    next_poll_at = $6,
		    completed_at = $7,
		    remote_response = $8::jsonb,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`,
		params.ID,
		params.RemoteJobID,
		params.PollURL,
		params.RemoteStatus,
		params.TerminalState,
		params.NextPollAt,
		params.CompletedAt,
		string(response),
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("update async task: %w", err)
	}

	return r.GetTaskByID(ctx, id)
}

func (r *SQLRepository) ListPolls(ctx context.Context, taskID int64, query ListQuery) (PollListResult, error) {
	q := normalizeListQuery(query)
	offset := (q.Page - 1) * q.PageSize

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM async_task_polls WHERE async_task_id = $1`, taskID); err != nil {
		return PollListResult{}, fmt.Errorf("count async task polls: %w", err)
	}

	rows := []PollRecord{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT id, async_task_id, polled_at, status_code, COALESCE(remote_status, '') AS remote_status,
		       COALESCE(response_body, '') AS response_body, COALESCE(error_message, '') AS error_message, duration_ms
		FROM async_task_polls
		WHERE async_task_id = $1
		ORDER BY polled_at DESC
		LIMIT $2 OFFSET $3
	`, taskID, q.PageSize, offset); err != nil {
		return PollListResult{}, fmt.Errorf("list async task polls: %w", err)
	}

	return PollListResult{
		Items:    rows,
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *SQLRepository) RecordPoll(ctx context.Context, input RecordPollInput) (PollRecord, error) {
	var record PollRecord
	if err := r.db.GetContext(ctx, &record, `
		INSERT INTO async_task_polls (
			async_task_id, polled_at, status_code, remote_status, response_body, error_message, duration_ms
		)
		VALUES ($1, NOW(), $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6)
		RETURNING id, async_task_id, polled_at, status_code, COALESCE(remote_status, '') AS remote_status,
		          COALESCE(response_body, '') AS response_body, COALESCE(error_message, '') AS error_message, duration_ms
	`, input.AsyncTaskID, input.StatusCode, input.RemoteStatus, input.ResponseBody, input.ErrorMessage, input.DurationMS); err != nil {
		return PollRecord{}, fmt.Errorf("record async task poll: %w", err)
	}
	return record, nil
}

func (r *SQLRepository) ListDueTasks(ctx context.Context, now time.Time, limit int) ([]Record, error) {
	if limit <= 0 {
		limit = 25
	}
	rows := []recordRow{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT a.id, a.uid::text AS uid, a.delivery_attempt_id, COALESCE(d.uid::text, '') AS delivery_uid,
		       COALESCE(d.request_id, 0) AS request_id, COALESCE(rq.uid::text, '') AS request_uid,
		       COALESCE(a.remote_job_id, '') AS remote_job_id, COALESCE(a.poll_url, '') AS poll_url,
		       COALESCE(a.remote_status, '') AS remote_status, COALESCE(a.terminal_state, '') AS terminal_state,
		       a.next_poll_at, a.completed_at, a.remote_response, a.created_at, a.updated_at
		FROM async_tasks a
		LEFT JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
		LEFT JOIN exchange_requests rq ON rq.id = d.request_id
		WHERE a.terminal_state IS NULL
		  AND (a.next_poll_at IS NULL OR a.next_poll_at <= $1)
		ORDER BY a.next_poll_at ASC NULLS FIRST, a.created_at ASC
		LIMIT $2
	`, now, limit); err != nil {
		return nil, fmt.Errorf("list due async tasks: %w", err)
	}
	return decodeRows(rows)
}

func resolveSortColumn(sortField string) string {
	switch sortField {
	case "uid":
		return "a.uid"
	case "deliveryUid":
		return "d.uid"
	case "requestUid":
		return "rq.uid"
	case "remoteJobId":
		return "a.remote_job_id"
	case "remoteStatus":
		return "a.remote_status"
	case "terminalState":
		return "a.terminal_state"
	case "nextPollAt":
		return "a.next_poll_at"
	case "completedAt":
		return "a.completed_at"
	case "updatedAt":
		return "a.updated_at"
	default:
		return "a.created_at"
	}
}

func decodeRows(rows []recordRow) ([]Record, error) {
	items := make([]Record, 0, len(rows))
	for _, row := range rows {
		item, err := decodeRow(row)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func decodeRow(row recordRow) (Record, error) {
	response, err := decodeJSONMap(row.RemoteResponse)
	if err != nil {
		return Record{}, fmt.Errorf("decode async remote response: %w", err)
	}
	return Record{
		ID:                row.ID,
		UID:               row.UID,
		DeliveryAttemptID: row.DeliveryAttemptID,
		DeliveryUID:       row.DeliveryUID,
		RequestID:         row.RequestID,
		RequestUID:        row.RequestUID,
		RemoteJobID:       row.RemoteJobID,
		PollURL:           row.PollURL,
		RemoteStatus:      row.RemoteStatus,
		TerminalState:     row.TerminalState,
		CurrentState:      deriveCurrentState(row.RemoteStatus, row.TerminalState),
		NextPollAt:        cloneTimePtr(row.NextPollAt),
		CompletedAt:       cloneTimePtr(row.CompletedAt),
		RemoteResponse:    response,
		CreatedAt:         row.CreatedAt,
		UpdatedAt:         row.UpdatedAt,
	}, nil
}

func decodeJSONMap(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var value map[string]any
	if err := json.Unmarshal(raw, &value); err != nil {
		return nil, err
	}
	if value == nil {
		return map[string]any{}, nil
	}
	return cloneJSONMap(value), nil
}

type memoryRepository struct {
	mu         sync.RWMutex
	nextID     int64
	nextPollID int64
	items      map[int64]Record
	polls      map[int64][]PollRecord
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		nextID:     1,
		nextPollID: 1,
		items:      map[int64]Record{},
		polls:      map[int64][]PollRecord{},
	}
}

func (r *memoryRepository) ListTasks(_ context.Context, query ListQuery) (ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := normalizeListQuery(query)
	filter := strings.ToLower(q.Filter)
	items := make([]Record, 0, len(r.items))
	for _, item := range r.items {
		if q.Status != "" && item.CurrentState != q.Status {
			continue
		}
		if filter != "" {
			searchable := strings.ToLower(strings.Join([]string{
				item.UID,
				item.DeliveryUID,
				item.RequestUID,
				item.RemoteJobID,
				item.RemoteStatus,
				item.TerminalState,
			}, " "))
			if !strings.Contains(searchable, filter) {
				continue
			}
		}
		items = append(items, cloneRecord(item))
	}

	sortRecords(items, q.SortField, q.SortOrder)
	start, end := paginate(len(items), q.Page, q.PageSize)

	return ListResult{
		Items:    slices.Clone(items[start:end]),
		Total:    len(items),
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *memoryRepository) GetTaskByID(_ context.Context, id int64) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	return cloneRecord(item), nil
}

func (r *memoryRepository) CreateTask(_ context.Context, params CreateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	id := r.nextID
	r.nextID++
	record := Record{
		ID:                id,
		UID:               params.UID,
		DeliveryAttemptID: params.DeliveryAttemptID,
		DeliveryUID:       fmt.Sprintf("delivery-%d", params.DeliveryAttemptID),
		RequestID:         params.DeliveryAttemptID,
		RequestUID:        fmt.Sprintf("request-%d", params.DeliveryAttemptID),
		RemoteJobID:       params.RemoteJobID,
		PollURL:           params.PollURL,
		RemoteStatus:      params.RemoteStatus,
		TerminalState:     params.TerminalState,
		CurrentState:      deriveCurrentState(params.RemoteStatus, params.TerminalState),
		NextPollAt:        cloneTimePtr(params.NextPollAt),
		CompletedAt:       cloneTimePtr(params.CompletedAt),
		RemoteResponse:    cloneJSONMap(params.RemoteResponse),
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	r.items[id] = record
	return cloneRecord(record), nil
}

func (r *memoryRepository) UpdateTask(_ context.Context, params UpdateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	record, ok := r.items[params.ID]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	record.RemoteJobID = params.RemoteJobID
	record.PollURL = params.PollURL
	record.RemoteStatus = params.RemoteStatus
	record.TerminalState = params.TerminalState
	record.CurrentState = deriveCurrentState(params.RemoteStatus, params.TerminalState)
	record.NextPollAt = cloneTimePtr(params.NextPollAt)
	record.CompletedAt = cloneTimePtr(params.CompletedAt)
	record.RemoteResponse = cloneJSONMap(params.RemoteResponse)
	record.UpdatedAt = time.Now().UTC()
	r.items[params.ID] = record
	return cloneRecord(record), nil
}

func (r *memoryRepository) ListPolls(_ context.Context, taskID int64, query ListQuery) (PollListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := normalizeListQuery(query)
	items := slices.Clone(r.polls[taskID])
	slices.SortFunc(items, func(a, b PollRecord) int {
		if a.PolledAt.Equal(b.PolledAt) {
			return 0
		}
		if a.PolledAt.After(b.PolledAt) {
			return -1
		}
		return 1
	})
	start, end := paginate(len(items), q.Page, q.PageSize)

	return PollListResult{
		Items:    slices.Clone(items[start:end]),
		Total:    len(items),
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *memoryRepository) RecordPoll(_ context.Context, input RecordPollInput) (PollRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[input.AsyncTaskID]; !ok {
		return PollRecord{}, sql.ErrNoRows
	}
	id := r.nextPollID
	r.nextPollID++
	record := PollRecord{
		ID:           id,
		AsyncTaskID:  input.AsyncTaskID,
		PolledAt:     time.Now().UTC(),
		StatusCode:   cloneIntPtr(input.StatusCode),
		RemoteStatus: input.RemoteStatus,
		ResponseBody: input.ResponseBody,
		ErrorMessage: input.ErrorMessage,
		DurationMS:   cloneIntPtr(input.DurationMS),
	}
	r.polls[input.AsyncTaskID] = append(r.polls[input.AsyncTaskID], record)
	return record, nil
}

func (r *memoryRepository) ListDueTasks(_ context.Context, now time.Time, limit int) ([]Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if limit <= 0 {
		limit = 25
	}
	items := make([]Record, 0, len(r.items))
	for _, item := range r.items {
		if item.TerminalState != "" {
			continue
		}
		if item.NextPollAt != nil && item.NextPollAt.After(now) {
			continue
		}
		items = append(items, cloneRecord(item))
	}
	slices.SortFunc(items, func(a, b Record) int {
		if a.NextPollAt == nil && b.NextPollAt == nil {
			return 0
		}
		if a.NextPollAt == nil {
			return -1
		}
		if b.NextPollAt == nil {
			return 1
		}
		if a.NextPollAt.Equal(*b.NextPollAt) {
			return 0
		}
		if a.NextPollAt.Before(*b.NextPollAt) {
			return -1
		}
		return 1
	})
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func sortRecords(items []Record, sortField string, sortOrder string) {
	less := func(a, b Record) bool {
		switch sortField {
		case "uid":
			return a.UID < b.UID
		case "deliveryUid":
			return a.DeliveryUID < b.DeliveryUID
		case "requestUid":
			return a.RequestUID < b.RequestUID
		case "remoteJobId":
			return a.RemoteJobID < b.RemoteJobID
		case "remoteStatus":
			return a.RemoteStatus < b.RemoteStatus
		case "terminalState":
			return a.TerminalState < b.TerminalState
		case "nextPollAt":
			return compareTimePtr(a.NextPollAt, b.NextPollAt)
		case "completedAt":
			return compareTimePtr(a.CompletedAt, b.CompletedAt)
		case "updatedAt":
			return a.UpdatedAt.Before(b.UpdatedAt)
		default:
			return a.CreatedAt.Before(b.CreatedAt)
		}
	}
	slices.SortFunc(items, func(a, b Record) int {
		switch {
		case less(a, b):
			return -1
		case less(b, a):
			return 1
		default:
			return 0
		}
	})
	if sortOrder == "desc" {
		slices.Reverse(items)
	}
}

func compareTimePtr(a *time.Time, b *time.Time) bool {
	if a == nil && b == nil {
		return false
	}
	if a == nil {
		return true
	}
	if b == nil {
		return false
	}
	return a.Before(*b)
}

func paginate(total int, page int, pageSize int) (int, int) {
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	return start, end
}

func cloneRecord(item Record) Record {
	item.NextPollAt = cloneTimePtr(item.NextPollAt)
	item.CompletedAt = cloneTimePtr(item.CompletedAt)
	item.RemoteResponse = cloneJSONMap(item.RemoteResponse)
	return item
}

func cloneJSONMap(input map[string]any) map[string]any {
	if input == nil {
		return map[string]any{}
	}
	cloned := make(map[string]any, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneIntPtr(value *int) *int {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func deriveCurrentState(remoteStatus string, terminalState string) string {
	if terminalState != "" {
		return terminalState
	}
	if remoteStatus == "" {
		return StatePending
	}
	switch strings.ToLower(strings.TrimSpace(remoteStatus)) {
	case StatePending, StatePolling, StateSucceeded, StateFailed:
		return strings.ToLower(strings.TrimSpace(remoteStatus))
	default:
		return StatePolling
	}
}

func newUID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(bytes[0:4]),
		hex.EncodeToString(bytes[4:6]),
		hex.EncodeToString(bytes[6:8]),
		hex.EncodeToString(bytes[8:10]),
		hex.EncodeToString(bytes[10:16]),
	)
}
