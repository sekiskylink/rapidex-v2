package worker

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

type row struct {
	ID              int64           `db:"id"`
	UID             string          `db:"uid"`
	WorkerType      string          `db:"worker_type"`
	WorkerName      string          `db:"worker_name"`
	Status          string          `db:"status"`
	StartedAt       time.Time       `db:"started_at"`
	StoppedAt       *time.Time      `db:"stopped_at"`
	LastHeartbeatAt *time.Time      `db:"last_heartbeat_at"`
	Meta            json.RawMessage `db:"meta"`
	CreatedAt       time.Time       `db:"created_at"`
	UpdatedAt       time.Time       `db:"updated_at"`
}

func normalizeQuery(query ListQuery) ListQuery {
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
	case "workerName", "workerType", "status", "startedAt", "lastHeartbeatAt", "updatedAt":
	default:
		sortField = "startedAt"
	}
	sortOrder := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if sortOrder != "asc" {
		sortOrder = "desc"
	}
	return ListQuery{Page: page, PageSize: pageSize, SortField: sortField, SortOrder: sortOrder}
}

func (r *SQLRepository) ListRuns(ctx context.Context, query ListQuery) (ListResult, error) {
	q := normalizeQuery(query)
	offset := (q.Page - 1) * q.PageSize

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM worker_runs`); err != nil {
		return ListResult{}, fmt.Errorf("count worker runs: %w", err)
	}

	rows := []row{}
	if err := r.db.SelectContext(ctx, &rows, fmt.Sprintf(`
		SELECT id, uid::text AS uid, worker_type, worker_name, status, started_at, stopped_at, last_heartbeat_at, meta, created_at, updated_at
		FROM worker_runs
		ORDER BY %s %s
		LIMIT $1 OFFSET $2
	`, resolveSortColumn(q.SortField), strings.ToUpper(q.SortOrder)), q.PageSize, offset); err != nil {
		return ListResult{}, fmt.Errorf("list worker runs: %w", err)
	}

	items, err := decodeRows(rows)
	if err != nil {
		return ListResult{}, err
	}
	return ListResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (r *SQLRepository) GetRunByID(ctx context.Context, id int64) (Record, error) {
	var item row
	if err := r.db.GetContext(ctx, &item, `
		SELECT id, uid::text AS uid, worker_type, worker_name, status, started_at, stopped_at, last_heartbeat_at, meta, created_at, updated_at
		FROM worker_runs
		WHERE id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("get worker run: %w", err)
	}
	return decodeRow(item)
}

func (r *SQLRepository) CreateRun(ctx context.Context, params CreateParams) (Record, error) {
	meta, err := json.Marshal(cloneJSONMap(params.Meta))
	if err != nil {
		return Record{}, fmt.Errorf("marshal worker meta: %w", err)
	}
	var id int64
	if err := r.db.GetContext(ctx, &id, `
		INSERT INTO worker_runs (uid, worker_type, worker_name, status, started_at, last_heartbeat_at, meta, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $5, $6::jsonb, NOW(), NOW())
		RETURNING id
	`, params.UID, params.WorkerType, params.WorkerName, params.Status, params.StartedAt, string(meta)); err != nil {
		return Record{}, fmt.Errorf("create worker run: %w", err)
	}
	return r.GetRunByID(ctx, id)
}

func (r *SQLRepository) UpdateRun(ctx context.Context, params UpdateParams) (Record, error) {
	meta, err := json.Marshal(cloneJSONMap(params.Meta))
	if err != nil {
		return Record{}, fmt.Errorf("marshal worker meta: %w", err)
	}
	var id int64
	if err := r.db.GetContext(ctx, &id, `
		UPDATE worker_runs
		SET status = $2,
		    stopped_at = $3,
		    last_heartbeat_at = $4,
		    meta = $5::jsonb,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, params.ID, params.Status, params.StoppedAt, params.LastHeartbeatAt, string(meta)); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("update worker run: %w", err)
	}
	return r.GetRunByID(ctx, id)
}

func resolveSortColumn(sortField string) string {
	switch sortField {
	case "workerName":
		return "worker_name"
	case "workerType":
		return "worker_type"
	case "status":
		return "status"
	case "lastHeartbeatAt":
		return "last_heartbeat_at"
	case "updatedAt":
		return "updated_at"
	default:
		return "started_at"
	}
}

func decodeRows(rows []row) ([]Record, error) {
	items := make([]Record, 0, len(rows))
	for _, item := range rows {
		decoded, err := decodeRow(item)
		if err != nil {
			return nil, err
		}
		items = append(items, decoded)
	}
	return items, nil
}

func decodeRow(item row) (Record, error) {
	meta := map[string]any{}
	if len(item.Meta) > 0 {
		if err := json.Unmarshal(item.Meta, &meta); err != nil {
			return Record{}, fmt.Errorf("decode worker meta: %w", err)
		}
	}
	return Record{
		ID:              item.ID,
		UID:             item.UID,
		WorkerType:      item.WorkerType,
		WorkerName:      item.WorkerName,
		Status:          item.Status,
		StartedAt:       item.StartedAt,
		StoppedAt:       cloneTimePtr(item.StoppedAt),
		LastHeartbeatAt: cloneTimePtr(item.LastHeartbeatAt),
		Meta:            cloneJSONMap(meta),
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}, nil
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

func (r *memoryRepository) ListRuns(_ context.Context, query ListQuery) (ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := normalizeQuery(query)
	items := make([]Record, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, cloneRecord(item))
	}
	slices.SortFunc(items, func(a, b Record) int {
		switch sortField := q.SortField; sortField {
		case "workerName":
			return strings.Compare(a.WorkerName, b.WorkerName)
		case "workerType":
			return strings.Compare(a.WorkerType, b.WorkerType)
		case "status":
			return strings.Compare(a.Status, b.Status)
		case "lastHeartbeatAt":
			return compareTime(a.LastHeartbeatAt, b.LastHeartbeatAt)
		case "updatedAt":
			return compareTime(&a.UpdatedAt, &b.UpdatedAt)
		default:
			return compareTime(&a.StartedAt, &b.StartedAt)
		}
	})
	if q.SortOrder == "desc" {
		slices.Reverse(items)
	}
	start, end := paginate(len(items), q.Page, q.PageSize)
	return ListResult{Items: slices.Clone(items[start:end]), Total: len(items), Page: q.Page, PageSize: q.PageSize}, nil
}

func (r *memoryRepository) GetRunByID(_ context.Context, id int64) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	return cloneRecord(item), nil
}

func (r *memoryRepository) CreateRun(_ context.Context, params CreateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextID
	r.nextID++
	now := time.Now().UTC()
	record := Record{
		ID:              id,
		UID:             params.UID,
		WorkerType:      params.WorkerType,
		WorkerName:      params.WorkerName,
		Status:          params.Status,
		StartedAt:       params.StartedAt,
		LastHeartbeatAt: &now,
		Meta:            cloneJSONMap(params.Meta),
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	r.items[id] = record
	return cloneRecord(record), nil
}

func (r *memoryRepository) UpdateRun(_ context.Context, params UpdateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	record, ok := r.items[params.ID]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	record.Status = params.Status
	record.StoppedAt = cloneTimePtr(params.StoppedAt)
	record.LastHeartbeatAt = cloneTimePtr(params.LastHeartbeatAt)
	record.Meta = cloneJSONMap(params.Meta)
	record.UpdatedAt = time.Now().UTC()
	r.items[params.ID] = record
	return cloneRecord(record), nil
}

func cloneRecord(item Record) Record {
	item.StoppedAt = cloneTimePtr(item.StoppedAt)
	item.LastHeartbeatAt = cloneTimePtr(item.LastHeartbeatAt)
	item.Meta = cloneJSONMap(item.Meta)
	return item
}

func cloneTimePtr(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
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

func compareTime(a *time.Time, b *time.Time) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}
	if a.Equal(*b) {
		return 0
	}
	if a.Before(*b) {
		return -1
	}
	return 1
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
