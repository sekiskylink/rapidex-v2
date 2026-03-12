package server

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/jmoiron/sqlx"
	"slices"
	"strings"
	"sync"
	"time"
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
	ID             int64           `db:"id"`
	UID            string          `db:"uid"`
	Name           string          `db:"name"`
	Code           string          `db:"code"`
	SystemType     string          `db:"system_type"`
	BaseURL        string          `db:"base_url"`
	EndpointType   string          `db:"endpoint_type"`
	HTTPMethod     string          `db:"http_method"`
	UseAsync       bool            `db:"use_async"`
	ParseResponses bool            `db:"parse_responses"`
	Headers        json.RawMessage `db:"headers"`
	URLParams      json.RawMessage `db:"url_params"`
	Suspended      bool            `db:"suspended"`
	CreatedAt      time.Time       `db:"created_at"`
	UpdatedAt      time.Time       `db:"updated_at"`
	CreatedBy      *int64          `db:"created_by"`
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
	case "id", "name", "code", "system_type", "base_url", "endpoint_type", "http_method", "use_async", "parse_responses", "suspended", "created_at", "updated_at":
	default:
		sortField = "name"
	}

	sortOrder := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if sortOrder != "desc" {
		sortOrder = "asc"
	}

	return ListQuery{
		Page:      page,
		PageSize:  pageSize,
		SortField: sortField,
		SortOrder: sortOrder,
		Filter:    strings.TrimSpace(query.Filter),
	}
}

func (r *SQLRepository) ListServers(ctx context.Context, query ListQuery) (ListResult, error) {
	q := normalizeListQuery(query)
	offset := (q.Page - 1) * q.PageSize

	whereClause := ""
	args := []any{}
	if q.Filter != "" {
		args = append(args, "%"+q.Filter+"%")
		whereClause = fmt.Sprintf(` WHERE name ILIKE $1 OR code ILIKE $1 OR system_type ILIKE $1 OR base_url ILIKE $1`)
	}

	var total int
	countQuery := `SELECT COUNT(*) FROM integration_servers` + whereClause
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return ListResult{}, fmt.Errorf("count integration servers: %w", err)
	}

	rows := []recordRow{}
	selectQuery := `
		SELECT id, uid, name, code, system_type, base_url, endpoint_type, http_method,
		       use_async, parse_responses, headers, url_params, suspended, created_at, updated_at, created_by
		FROM integration_servers
	` + whereClause + fmt.Sprintf(" ORDER BY %s %s", q.SortField, strings.ToUpper(q.SortOrder))

	args = append(args, q.PageSize, offset)
	selectQuery += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))
	if err := r.db.SelectContext(ctx, &rows, selectQuery, args...); err != nil {
		return ListResult{}, fmt.Errorf("list integration servers: %w", err)
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

func (r *SQLRepository) GetServerByID(ctx context.Context, id int64) (Record, error) {
	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT id, uid, name, code, system_type, base_url, endpoint_type, http_method,
		       use_async, parse_responses, headers, url_params, suspended, created_at, updated_at, created_by
		FROM integration_servers
		WHERE id = $1
	`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("get integration server: %w", err)
	}
	return decodeRow(row)
}

func (r *SQLRepository) CreateServer(ctx context.Context, params CreateParams) (Record, error) {
	headers, err := json.Marshal(cloneStringMap(params.Headers))
	if err != nil {
		return Record{}, fmt.Errorf("marshal headers: %w", err)
	}
	urlParams, err := json.Marshal(cloneStringMap(params.URLParams))
	if err != nil {
		return Record{}, fmt.Errorf("marshal url params: %w", err)
	}

	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		INSERT INTO integration_servers (
			uid, name, code, system_type, base_url, endpoint_type, http_method,
			use_async, parse_responses, headers, url_params, suspended, created_at, updated_at, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, $12, NOW(), NOW(), $13)
		RETURNING id, uid, name, code, system_type, base_url, endpoint_type, http_method,
		          use_async, parse_responses, headers, url_params, suspended, created_at, updated_at, created_by
	`,
		params.UID,
		params.Name,
		params.Code,
		params.SystemType,
		params.BaseURL,
		params.EndpointType,
		params.HTTPMethod,
		params.UseAsync,
		params.ParseResponses,
		string(headers),
		string(urlParams),
		params.Suspended,
		params.CreatedBy,
	); err != nil {
		return Record{}, fmt.Errorf("create integration server: %w", err)
	}

	return decodeRow(row)
}

func (r *SQLRepository) UpdateServer(ctx context.Context, params UpdateParams) (Record, error) {
	headers, err := json.Marshal(cloneStringMap(params.Headers))
	if err != nil {
		return Record{}, fmt.Errorf("marshal headers: %w", err)
	}
	urlParams, err := json.Marshal(cloneStringMap(params.URLParams))
	if err != nil {
		return Record{}, fmt.Errorf("marshal url params: %w", err)
	}

	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		UPDATE integration_servers
		SET name = $2,
		    code = $3,
		    system_type = $4,
		    base_url = $5,
		    endpoint_type = $6,
		    http_method = $7,
		    use_async = $8,
		    parse_responses = $9,
		    headers = $10::jsonb,
		    url_params = $11::jsonb,
		    suspended = $12,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id, uid, name, code, system_type, base_url, endpoint_type, http_method,
		          use_async, parse_responses, headers, url_params, suspended, created_at, updated_at, created_by
	`,
		params.ID,
		params.Name,
		params.Code,
		params.SystemType,
		params.BaseURL,
		params.EndpointType,
		params.HTTPMethod,
		params.UseAsync,
		params.ParseResponses,
		string(headers),
		string(urlParams),
		params.Suspended,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("update integration server: %w", err)
	}

	return decodeRow(row)
}

func (r *SQLRepository) DeleteServer(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM integration_servers WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete integration server: %w", err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete integration server rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}
	return nil
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
	headers, err := decodeJSONMap(row.Headers)
	if err != nil {
		return Record{}, fmt.Errorf("decode integration server headers: %w", err)
	}
	urlParams, err := decodeJSONMap(row.URLParams)
	if err != nil {
		return Record{}, fmt.Errorf("decode integration server url params: %w", err)
	}

	return Record{
		ID:             row.ID,
		UID:            row.UID,
		Name:           row.Name,
		Code:           row.Code,
		SystemType:     row.SystemType,
		BaseURL:        row.BaseURL,
		EndpointType:   row.EndpointType,
		HTTPMethod:     row.HTTPMethod,
		UseAsync:       row.UseAsync,
		ParseResponses: row.ParseResponses,
		Headers:        headers,
		URLParams:      urlParams,
		Suspended:      row.Suspended,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		CreatedBy:      row.CreatedBy,
	}, nil
}

func decodeJSONMap(raw json.RawMessage) (map[string]string, error) {
	if len(raw) == 0 {
		return map[string]string{}, nil
	}
	var parsed map[string]string
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	return cloneStringMap(parsed), nil
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

func (r *memoryRepository) ListServers(_ context.Context, query ListQuery) (ListResult, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	q := normalizeListQuery(query)
	filter := strings.ToLower(q.Filter)

	items := make([]Record, 0, len(r.items))
	for _, item := range r.items {
		if filter != "" {
			searchable := strings.ToLower(strings.Join([]string{item.Name, item.Code, item.SystemType, item.BaseURL}, " "))
			if !strings.Contains(searchable, filter) {
				continue
			}
		}
		items = append(items, cloneRecord(item))
	}

	slices.SortFunc(items, func(a, b Record) int {
		switch q.SortField {
		case "id":
			if a.ID < b.ID {
				return -1
			}
			if a.ID > b.ID {
				return 1
			}
			return 0
		case "code":
			return strings.Compare(a.Code, b.Code)
		case "system_type":
			return strings.Compare(a.SystemType, b.SystemType)
		case "base_url":
			return strings.Compare(a.BaseURL, b.BaseURL)
		case "updated_at":
			if a.UpdatedAt.Before(b.UpdatedAt) {
				return -1
			}
			if a.UpdatedAt.After(b.UpdatedAt) {
				return 1
			}
			return 0
		case "created_at":
			if a.CreatedAt.Before(b.CreatedAt) {
				return -1
			}
			if a.CreatedAt.After(b.CreatedAt) {
				return 1
			}
			return 0
		default:
			return strings.Compare(a.Name, b.Name)
		}
	})
	if q.SortOrder == "desc" {
		slices.Reverse(items)
	}

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
		Items:    append([]Record(nil), items[start:end]...),
		Total:    total,
		Page:     q.Page,
		PageSize: q.PageSize,
	}, nil
}

func (r *memoryRepository) GetServerByID(_ context.Context, id int64) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	item, ok := r.items[id]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	return cloneRecord(item), nil
}

func (r *memoryRepository) CreateServer(_ context.Context, params CreateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range r.items {
		if strings.EqualFold(item.Code, params.Code) {
			return Record{}, fmt.Errorf("create integration server: duplicate code")
		}
	}

	now := time.Now().UTC()
	record := Record{
		ID:             r.nextID,
		UID:            params.UID,
		Name:           params.Name,
		Code:           params.Code,
		SystemType:     params.SystemType,
		BaseURL:        params.BaseURL,
		EndpointType:   params.EndpointType,
		HTTPMethod:     params.HTTPMethod,
		UseAsync:       params.UseAsync,
		ParseResponses: params.ParseResponses,
		Headers:        cloneStringMap(params.Headers),
		URLParams:      cloneStringMap(params.URLParams),
		Suspended:      params.Suspended,
		CreatedAt:      now,
		UpdatedAt:      now,
		CreatedBy:      params.CreatedBy,
	}
	r.items[record.ID] = record
	r.nextID++
	return cloneRecord(record), nil
}

func (r *memoryRepository) UpdateServer(_ context.Context, params UpdateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.items[params.ID]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	for _, item := range r.items {
		if item.ID != params.ID && strings.EqualFold(item.Code, params.Code) {
			return Record{}, fmt.Errorf("update integration server: duplicate code")
		}
	}

	existing.Name = params.Name
	existing.Code = params.Code
	existing.SystemType = params.SystemType
	existing.BaseURL = params.BaseURL
	existing.EndpointType = params.EndpointType
	existing.HTTPMethod = params.HTTPMethod
	existing.UseAsync = params.UseAsync
	existing.ParseResponses = params.ParseResponses
	existing.Headers = cloneStringMap(params.Headers)
	existing.URLParams = cloneStringMap(params.URLParams)
	existing.Suspended = params.Suspended
	existing.UpdatedAt = time.Now().UTC()
	r.items[params.ID] = existing
	return cloneRecord(existing), nil
}

func (r *memoryRepository) DeleteServer(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return sql.ErrNoRows
	}
	delete(r.items, id)
	return nil
}

func cloneStringMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneRecord(input Record) Record {
	input.Headers = cloneStringMap(input.Headers)
	input.URLParams = cloneStringMap(input.URLParams)
	return input
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
