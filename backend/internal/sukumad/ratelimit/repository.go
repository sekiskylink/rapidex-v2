package ratelimit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
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
	case "name", "scopeType", "scopeRef", "rps", "burst", "maxConcurrency", "isActive", "updatedAt":
	default:
		sortField = "name"
	}
	sortOrder := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if sortOrder != "desc" {
		sortOrder = "asc"
	}
	return ListQuery{Page: page, PageSize: pageSize, SortField: sortField, SortOrder: sortOrder}
}

func (r *SQLRepository) ListPolicies(ctx context.Context, query ListQuery) (ListResult, error) {
	q := normalizeQuery(query)
	offset := (q.Page - 1) * q.PageSize

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM rate_limit_policies`); err != nil {
		return ListResult{}, fmt.Errorf("count rate limit policies: %w", err)
	}

	items := []Policy{}
	if err := r.db.SelectContext(ctx, &items, fmt.Sprintf(`
		SELECT id, uid::text AS uid, name, scope_type, COALESCE(scope_ref, '') AS scope_ref, rps, burst, max_concurrency, timeout_ms, is_active, created_at, updated_at
		FROM rate_limit_policies
		ORDER BY %s %s
		LIMIT $1 OFFSET $2
	`, resolveSortColumn(q.SortField), strings.ToUpper(q.SortOrder)), q.PageSize, offset); err != nil {
		return ListResult{}, fmt.Errorf("list rate limit policies: %w", err)
	}
	return ListResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (r *SQLRepository) GetPolicyByID(ctx context.Context, id int64) (Policy, error) {
	var item Policy
	if err := r.db.GetContext(ctx, &item, `
		SELECT id, uid::text AS uid, name, scope_type, COALESCE(scope_ref, '') AS scope_ref, rps, burst, max_concurrency, timeout_ms, is_active, created_at, updated_at
		FROM rate_limit_policies
		WHERE id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Policy{}, sql.ErrNoRows
		}
		return Policy{}, fmt.Errorf("get rate limit policy: %w", err)
	}
	return item, nil
}

func (r *SQLRepository) CreatePolicy(ctx context.Context, params CreateParams) (Policy, error) {
	var id int64
	if err := r.db.GetContext(ctx, &id, `
		INSERT INTO rate_limit_policies (uid, name, scope_type, scope_ref, rps, burst, max_concurrency, timeout_ms, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, NULLIF($4, ''), $5, $6, $7, $8, $9, NOW(), NOW())
		RETURNING id
	`,
		params.UID,
		params.Name,
		params.ScopeType,
		params.ScopeRef,
		params.RPS,
		params.Burst,
		params.MaxConcurrency,
		params.TimeoutMS,
		params.IsActive,
	); err != nil {
		return Policy{}, fmt.Errorf("create rate limit policy: %w", err)
	}
	return r.GetPolicyByID(ctx, id)
}

func (r *SQLRepository) FindActivePolicy(ctx context.Context, scopeType string, scopeRef string) (Policy, bool, error) {
	var item Policy
	if err := r.db.GetContext(ctx, &item, `
		SELECT id, uid::text AS uid, name, scope_type, COALESCE(scope_ref, '') AS scope_ref, rps, burst, max_concurrency, timeout_ms, is_active, created_at, updated_at
		FROM rate_limit_policies
		WHERE is_active = TRUE
		  AND scope_type = $1
		  AND (COALESCE(scope_ref, '') = $2 OR COALESCE(scope_ref, '') = '')
		ORDER BY CASE WHEN COALESCE(scope_ref, '') = $2 THEN 0 ELSE 1 END, created_at ASC
		LIMIT 1
	`, scopeType, scopeRef); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Policy{}, false, nil
		}
		return Policy{}, false, fmt.Errorf("find active rate limit policy: %w", err)
	}
	return item, true, nil
}

func resolveSortColumn(sortField string) string {
	switch sortField {
	case "scopeType":
		return "scope_type"
	case "scopeRef":
		return "scope_ref"
	case "rps":
		return "rps"
	case "burst":
		return "burst"
	case "maxConcurrency":
		return "max_concurrency"
	case "isActive":
		return "is_active"
	case "updatedAt":
		return "updated_at"
	default:
		return "name"
	}
}

type memoryRepository struct {
	mu     sync.RWMutex
	nextID int64
	items  map[int64]Policy
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{nextID: 1, items: map[int64]Policy{}}
}

func (r *memoryRepository) ListPolicies(_ context.Context, query ListQuery) (ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	q := normalizeQuery(query)
	items := make([]Policy, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, item)
	}
	slices.SortFunc(items, func(a, b Policy) int {
		return strings.Compare(a.Name, b.Name)
	})
	if q.SortOrder == "desc" {
		slices.Reverse(items)
	}
	start, end := paginate(len(items), q.Page, q.PageSize)
	return ListResult{Items: slices.Clone(items[start:end]), Total: len(items), Page: q.Page, PageSize: q.PageSize}, nil
}

func (r *memoryRepository) GetPolicyByID(_ context.Context, id int64) (Policy, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[id]
	if !ok {
		return Policy{}, sql.ErrNoRows
	}
	return item, nil
}

func (r *memoryRepository) CreatePolicy(_ context.Context, params CreateParams) (Policy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	id := r.nextID
	r.nextID++
	now := time.Now().UTC()
	item := Policy{
		ID:             id,
		UID:            params.UID,
		Name:           params.Name,
		ScopeType:      params.ScopeType,
		ScopeRef:       params.ScopeRef,
		RPS:            params.RPS,
		Burst:          params.Burst,
		MaxConcurrency: params.MaxConcurrency,
		TimeoutMS:      params.TimeoutMS,
		IsActive:       params.IsActive,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	r.items[id] = item
	return item, nil
}

func (r *memoryRepository) FindActivePolicy(_ context.Context, scopeType string, scopeRef string) (Policy, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, item := range r.items {
		if !item.IsActive || item.ScopeType != scopeType {
			continue
		}
		if item.ScopeRef == scopeRef || item.ScopeRef == "" {
			return item, true, nil
		}
	}
	return Policy{}, false, nil
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
