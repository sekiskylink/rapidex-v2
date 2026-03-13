package request

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
	ID                     int64           `db:"id"`
	UID                    string          `db:"uid"`
	SourceSystem           string          `db:"source_system"`
	DestinationServerID    int64           `db:"destination_server_id"`
	DestinationServerName  string          `db:"destination_server_name"`
	BatchID                string          `db:"batch_id"`
	CorrelationID          string          `db:"correlation_id"`
	IdempotencyKey         string          `db:"idempotency_key"`
	PayloadBody            string          `db:"payload_body"`
	PayloadFormat          string          `db:"payload_format"`
	URLSuffix              string          `db:"url_suffix"`
	Status                 string          `db:"status"`
	Extras                 json.RawMessage `db:"extras"`
	CreatedAt              time.Time       `db:"created_at"`
	UpdatedAt              time.Time       `db:"updated_at"`
	CreatedBy              *int64          `db:"created_by"`
	LatestDeliveryID       *int64          `db:"latest_delivery_id"`
	LatestDeliveryUID      string          `db:"latest_delivery_uid"`
	LatestDeliveryStatus   string          `db:"latest_delivery_status"`
	LatestAsyncTaskID      *int64          `db:"latest_async_task_id"`
	LatestAsyncTaskUID     string          `db:"latest_async_task_uid"`
	LatestAsyncState       string          `db:"latest_async_state"`
	LatestAsyncRemoteJobID string          `db:"latest_async_remote_job_id"`
	LatestAsyncPollURL     string          `db:"latest_async_poll_url"`
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
	case "uid", "sourceSystem", "destinationServerName", "correlationId", "status", "createdAt", "updatedAt":
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

func (r *SQLRepository) ListRequests(ctx context.Context, query ListQuery) (ListResult, error) {
	q := normalizeListQuery(query)
	offset := (q.Page - 1) * q.PageSize

	conditions := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if q.Filter != "" {
		args = append(args, "%"+q.Filter+"%")
		needle := fmt.Sprintf("$%d", len(args))
		conditions = append(conditions, `(
			r.uid::text ILIKE `+needle+` OR
			COALESCE(r.source_system, '') ILIKE `+needle+` OR
			COALESCE(r.correlation_id, '') ILIKE `+needle+` OR
			COALESCE(r.batch_id, '') ILIKE `+needle+` OR
			COALESCE(r.idempotency_key, '') ILIKE `+needle+` OR
			COALESCE(r.url_suffix, '') ILIKE `+needle+` OR
			COALESCE(s.name, '') ILIKE `+needle+` OR
			COALESCE(s.code, '') ILIKE `+needle+`
		)`)
	}
	if q.Status != "" {
		args = append(args, q.Status)
		conditions = append(conditions, fmt.Sprintf("r.status = $%d", len(args)))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	baseFrom := `
		FROM exchange_requests r
		LEFT JOIN integration_servers s ON s.id = r.destination_server_id
		LEFT JOIN LATERAL (
			SELECT d.id,
			       d.uid::text AS uid,
			       d.status
			FROM delivery_attempts d
			WHERE d.request_id = r.id
			ORDER BY d.attempt_number DESC, d.created_at DESC
			LIMIT 1
		) ld ON TRUE
		LEFT JOIN async_tasks a ON a.delivery_attempt_id = ld.id
	`

	var total int
	countQuery := `SELECT COUNT(*) ` + baseFrom + whereClause
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return ListResult{}, fmt.Errorf("count exchange requests: %w", err)
	}

	selectArgs := append([]any{}, args...)
	selectArgs = append(selectArgs, q.PageSize, offset)
	querySQL := `
		SELECT r.id, r.uid::text AS uid, r.source_system, r.destination_server_id,
		       COALESCE(s.name, '') AS destination_server_name,
		       r.batch_id, r.correlation_id, r.idempotency_key,
		       r.payload_body, r.payload_format, r.url_suffix, r.status,
		       r.extras, r.created_at, r.updated_at, r.created_by,
		       ld.id AS latest_delivery_id,
		       COALESCE(ld.uid, '') AS latest_delivery_uid,
		       COALESCE(ld.status, '') AS latest_delivery_status,
		       a.id AS latest_async_task_id,
		       COALESCE(a.uid::text, '') AS latest_async_task_uid,
		       COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN '' ELSE a.remote_status END) AS latest_async_state,
		       COALESCE(a.remote_job_id, '') AS latest_async_remote_job_id,
		       COALESCE(a.poll_url, '') AS latest_async_poll_url
	` + baseFrom + whereClause + fmt.Sprintf(
		" ORDER BY %s %s LIMIT $%d OFFSET $%d",
		resolveSortColumn(q.SortField),
		strings.ToUpper(q.SortOrder),
		len(selectArgs)-1,
		len(selectArgs),
	)

	rows := []recordRow{}
	if err := r.db.SelectContext(ctx, &rows, querySQL, selectArgs...); err != nil {
		return ListResult{}, fmt.Errorf("list exchange requests: %w", err)
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

func (r *SQLRepository) GetRequestByID(ctx context.Context, id int64) (Record, error) {
	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT r.id, r.uid::text AS uid, r.source_system, r.destination_server_id,
		       COALESCE(s.name, '') AS destination_server_name,
		       r.batch_id, r.correlation_id, r.idempotency_key,
		       r.payload_body, r.payload_format, r.url_suffix, r.status,
		       r.extras, r.created_at, r.updated_at, r.created_by,
		       ld.id AS latest_delivery_id,
		       COALESCE(ld.uid, '') AS latest_delivery_uid,
		       COALESCE(ld.status, '') AS latest_delivery_status,
		       a.id AS latest_async_task_id,
		       COALESCE(a.uid::text, '') AS latest_async_task_uid,
		       COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN '' ELSE a.remote_status END) AS latest_async_state,
		       COALESCE(a.remote_job_id, '') AS latest_async_remote_job_id,
		       COALESCE(a.poll_url, '') AS latest_async_poll_url
		FROM exchange_requests r
		LEFT JOIN integration_servers s ON s.id = r.destination_server_id
		LEFT JOIN LATERAL (
			SELECT d.id,
			       d.uid::text AS uid,
			       d.status
			FROM delivery_attempts d
			WHERE d.request_id = r.id
			ORDER BY d.attempt_number DESC, d.created_at DESC
			LIMIT 1
		) ld ON TRUE
		LEFT JOIN async_tasks a ON a.delivery_attempt_id = ld.id
		WHERE r.id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("get exchange request: %w", err)
	}
	return decodeRow(row)
}

func (r *SQLRepository) CreateRequest(ctx context.Context, params CreateParams) (Record, error) {
	extras, err := json.Marshal(cloneExtras(params.Extras))
	if err != nil {
		return Record{}, fmt.Errorf("marshal request extras: %w", err)
	}

	var id int64
	if err := r.db.GetContext(ctx, &id, `
		INSERT INTO exchange_requests (
			uid, source_system, destination_server_id, batch_id, correlation_id, idempotency_key,
			payload_body, payload_format, url_suffix, status, extras, created_at, updated_at, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11::jsonb, NOW(), NOW(), $12)
		RETURNING id
	`,
		params.UID,
		params.SourceSystem,
		params.DestinationServerID,
		nullIfEmpty(params.BatchID),
		nullIfEmpty(params.CorrelationID),
		nullIfEmpty(params.IdempotencyKey),
		params.PayloadBody,
		params.PayloadFormat,
		nullIfEmpty(params.URLSuffix),
		params.Status,
		string(extras),
		params.CreatedBy,
	); err != nil {
		return Record{}, fmt.Errorf("create exchange request: %w", err)
	}

	return r.GetRequestByID(ctx, id)
}

func (r *SQLRepository) UpdateRequestStatus(ctx context.Context, id int64, status string) (Record, error) {
	var updatedID int64
	if err := r.db.GetContext(ctx, &updatedID, `
		UPDATE exchange_requests
		SET status = $2,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, id, status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("update exchange request status: %w", err)
	}
	return r.GetRequestByID(ctx, updatedID)
}

func resolveSortColumn(sortField string) string {
	switch sortField {
	case "uid":
		return "r.uid"
	case "sourceSystem":
		return "r.source_system"
	case "destinationServerName":
		return "s.name"
	case "correlationId":
		return "r.correlation_id"
	case "status":
		return "r.status"
	case "updatedAt":
		return "r.updated_at"
	default:
		return "r.created_at"
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
	extras, err := decodeExtras(row.Extras)
	if err != nil {
		return Record{}, fmt.Errorf("decode exchange request extras: %w", err)
	}
	return Record{
		ID:                     row.ID,
		UID:                    row.UID,
		SourceSystem:           row.SourceSystem,
		DestinationServerID:    row.DestinationServerID,
		DestinationServerName:  row.DestinationServerName,
		BatchID:                row.BatchID,
		CorrelationID:          row.CorrelationID,
		IdempotencyKey:         row.IdempotencyKey,
		PayloadBody:            row.PayloadBody,
		PayloadFormat:          row.PayloadFormat,
		URLSuffix:              row.URLSuffix,
		Status:                 row.Status,
		Extras:                 extras,
		CreatedAt:              row.CreatedAt,
		UpdatedAt:              row.UpdatedAt,
		CreatedBy:              row.CreatedBy,
		Payload:                json.RawMessage(row.PayloadBody),
		LatestDeliveryID:       cloneInt64Ptr(row.LatestDeliveryID),
		LatestDeliveryUID:      row.LatestDeliveryUID,
		LatestDeliveryStatus:   row.LatestDeliveryStatus,
		LatestAsyncTaskID:      cloneInt64Ptr(row.LatestAsyncTaskID),
		LatestAsyncTaskUID:     row.LatestAsyncTaskUID,
		LatestAsyncState:       row.LatestAsyncState,
		LatestAsyncRemoteJobID: row.LatestAsyncRemoteJobID,
		LatestAsyncPollURL:     row.LatestAsyncPollURL,
		AwaitingAsync:          row.LatestAsyncTaskID != nil && row.LatestAsyncState != "" && row.LatestAsyncState != StatusCompleted && row.LatestAsyncState != StatusFailed,
	}, nil
}

func decodeExtras(raw json.RawMessage) (map[string]any, error) {
	if len(raw) == 0 {
		return map[string]any{}, nil
	}
	var parsed map[string]any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	if parsed == nil {
		return map[string]any{}, nil
	}
	return cloneExtras(parsed), nil
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

func (r *memoryRepository) ListRequests(_ context.Context, query ListQuery) (ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := normalizeListQuery(query)
	filter := strings.ToLower(q.Filter)

	items := make([]Record, 0, len(r.items))
	for _, item := range r.items {
		if q.Status != "" && item.Status != q.Status {
			continue
		}
		if filter != "" {
			searchable := strings.ToLower(strings.Join([]string{
				item.UID,
				item.SourceSystem,
				item.DestinationServerName,
				item.CorrelationID,
				item.BatchID,
				item.IdempotencyKey,
				item.URLSuffix,
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
		case "sourceSystem":
			cmp = strings.Compare(a.SourceSystem, b.SourceSystem)
		case "destinationServerName":
			cmp = strings.Compare(a.DestinationServerName, b.DestinationServerName)
		case "correlationId":
			cmp = strings.Compare(a.CorrelationID, b.CorrelationID)
		case "status":
			cmp = strings.Compare(a.Status, b.Status)
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

func (r *memoryRepository) GetRequestByID(_ context.Context, id int64) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	return cloneRecord(item), nil
}

func (r *memoryRepository) CreateRequest(_ context.Context, params CreateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	id := r.nextID
	r.nextID++
	now := time.Now().UTC()
	record := Record{
		ID:                    id,
		UID:                   params.UID,
		SourceSystem:          params.SourceSystem,
		DestinationServerID:   params.DestinationServerID,
		DestinationServerName: fmt.Sprintf("Server #%d", params.DestinationServerID),
		BatchID:               params.BatchID,
		CorrelationID:         params.CorrelationID,
		IdempotencyKey:        params.IdempotencyKey,
		PayloadBody:           params.PayloadBody,
		PayloadFormat:         params.PayloadFormat,
		URLSuffix:             params.URLSuffix,
		Status:                params.Status,
		Extras:                cloneExtras(params.Extras),
		CreatedAt:             now,
		UpdatedAt:             now,
		CreatedBy:             params.CreatedBy,
		Payload:               json.RawMessage(params.PayloadBody),
	}
	r.items[id] = record
	return cloneRecord(record), nil
}

func (r *memoryRepository) UpdateRequestStatus(_ context.Context, id int64, status string) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[id]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	item.Status = status
	item.UpdatedAt = time.Now().UTC()
	r.items[id] = item
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

func cloneRecord(input Record) Record {
	input.Extras = cloneExtras(input.Extras)
	input.Payload = append(json.RawMessage(nil), input.Payload...)
	input.LatestDeliveryID = cloneInt64Ptr(input.LatestDeliveryID)
	input.LatestAsyncTaskID = cloneInt64Ptr(input.LatestAsyncTaskID)
	return input
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneExtras(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func nullIfEmpty(value string) any {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return trimmed
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
