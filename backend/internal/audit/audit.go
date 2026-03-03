package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

type Event struct {
	Timestamp   time.Time
	ActorUserID *int64
	Action      string
	EntityType  string
	EntityID    *string
	Metadata    map[string]any
}

type Repository interface {
	Insert(ctx context.Context, event Event) error
	List(ctx context.Context, filter ListFilter) (ListResult, error)
}

type Record struct {
	ID         int64           `db:"id" json:"id"`
	Timestamp  time.Time       `db:"timestamp" json:"timestamp"`
	ActorUser  *int64          `db:"actor_user_id" json:"actorUserId,omitempty"`
	Action     string          `db:"action" json:"action"`
	EntityType string          `db:"entity_type" json:"entityType,omitempty"`
	EntityID   *string         `db:"entity_id" json:"entityId,omitempty"`
	Metadata   json.RawMessage `db:"metadata_json" json:"metadata"`
}

type ListFilter struct {
	Page        int
	PageSize    int
	SortField   string
	SortOrder   string
	Action      string
	ActorUserID *int64
	DateFrom    *time.Time
	DateTo      *time.Time
}

type ListResult struct {
	Items    []Record
	Total    int
	Page     int
	PageSize int
}

type SQLRepository struct {
	db *sqlx.DB
}

func NewSQLRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

func (r *SQLRepository) Insert(ctx context.Context, event Event) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return fmt.Errorf("marshal audit metadata: %w", err)
	}

	_, err = r.db.ExecContext(ctx, `
		INSERT INTO audit_logs (timestamp, actor_user_id, action, entity_type, entity_id, metadata_json, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, event.Timestamp, event.ActorUserID, event.Action, event.EntityType, event.EntityID, metadata, event.Timestamp)
	if err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}

func normalizeListFilter(filter ListFilter) ListFilter {
	page := filter.Page
	if page <= 0 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 25
	}
	sortField := strings.TrimSpace(filter.SortField)
	switch sortField {
	case "timestamp", "action", "id", "actor_user_id", "entity_type", "entity_id":
	default:
		sortField = "timestamp"
	}
	sortOrder := strings.ToLower(strings.TrimSpace(filter.SortOrder))
	if sortOrder != "asc" {
		sortOrder = "desc"
	}

	return ListFilter{
		Page:        page,
		PageSize:    pageSize,
		SortField:   sortField,
		SortOrder:   sortOrder,
		Action:      strings.TrimSpace(filter.Action),
		ActorUserID: filter.ActorUserID,
		DateFrom:    filter.DateFrom,
		DateTo:      filter.DateTo,
	}
}

func (r *SQLRepository) List(ctx context.Context, filter ListFilter) (ListResult, error) {
	f := normalizeListFilter(filter)
	offset := (f.Page - 1) * f.PageSize

	conditions := []string{}
	args := []any{}
	if f.Action != "" {
		args = append(args, "%"+f.Action+"%")
		conditions = append(conditions, fmt.Sprintf("action ILIKE $%d", len(args)))
	}
	if f.ActorUserID != nil {
		args = append(args, *f.ActorUserID)
		conditions = append(conditions, fmt.Sprintf("actor_user_id = $%d", len(args)))
	}
	if f.DateFrom != nil {
		args = append(args, *f.DateFrom)
		conditions = append(conditions, fmt.Sprintf("timestamp >= $%d", len(args)))
	}
	if f.DateTo != nil {
		args = append(args, *f.DateTo)
		conditions = append(conditions, fmt.Sprintf("timestamp <= $%d", len(args)))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	total := 0
	countQuery := `SELECT COUNT(*) FROM audit_logs` + whereClause
	if err := r.db.GetContext(ctx, &total, countQuery, args...); err != nil {
		return ListResult{}, fmt.Errorf("count audit logs: %w", err)
	}

	query := `
		SELECT id, timestamp, actor_user_id, action, entity_type, entity_id, metadata_json
		FROM audit_logs
	`
	query += whereClause
	query += fmt.Sprintf(" ORDER BY %s %s", f.SortField, strings.ToUpper(f.SortOrder))
	args = append(args, f.PageSize, offset)
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	records := []Record{}
	if err := r.db.SelectContext(ctx, &records, query, args...); err != nil {
		return ListResult{}, fmt.Errorf("list audit logs: %w", err)
	}
	return ListResult{
		Items:    records,
		Total:    total,
		Page:     f.Page,
		PageSize: f.PageSize,
	}, nil
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Log(ctx context.Context, event Event) error {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now().UTC()
	}
	if event.Metadata == nil {
		event.Metadata = map[string]any{}
	}
	return s.repo.Insert(ctx, event)
}

func (s *Service) List(ctx context.Context, filter ListFilter) (ListResult, error) {
	return s.repo.List(ctx, filter)
}
