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
	List(ctx context.Context, filter ListFilter) ([]Record, error)
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
	Limit  int
	Offset int
	Action string
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

func (r *SQLRepository) List(ctx context.Context, filter ListFilter) ([]Record, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	args := []any{limit, offset}
	query := `
		SELECT id, timestamp, actor_user_id, action, entity_type, entity_id, metadata_json
		FROM audit_logs
	`
	if strings.TrimSpace(filter.Action) != "" {
		query += " WHERE action = $3"
		args = append(args, strings.TrimSpace(filter.Action))
	}
	query += " ORDER BY timestamp DESC LIMIT $1 OFFSET $2"

	records := []Record{}
	if err := r.db.SelectContext(ctx, &records, query, args...); err != nil {
		return nil, fmt.Errorf("list audit logs: %w", err)
	}
	return records, nil
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

func (s *Service) List(ctx context.Context, filter ListFilter) ([]Record, error) {
	return s.repo.List(ctx, filter)
}
