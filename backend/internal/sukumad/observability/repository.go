package observability

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"basepro/backend/internal/sukumad/ratelimit"
	"basepro/backend/internal/sukumad/worker"
	"github.com/jmoiron/sqlx"
)

type Repository struct {
	db      *sqlx.DB
	workers interface {
		ListRuns(context.Context, worker.ListQuery) (worker.ListResult, error)
		GetRun(context.Context, int64) (worker.Record, error)
	}
	rateLimits interface {
		ListPolicies(context.Context, ratelimit.ListQuery) (ratelimit.ListResult, error)
	}
}

func NewRepository(
	db *sqlx.DB,
	workers interface {
		ListRuns(context.Context, worker.ListQuery) (worker.ListResult, error)
		GetRun(context.Context, int64) (worker.Record, error)
	},
	rateLimits interface {
		ListPolicies(context.Context, ratelimit.ListQuery) (ratelimit.ListResult, error)
	},
) *Repository {
	return &Repository{db: db, workers: workers, rateLimits: rateLimits}
}

type eventRow struct {
	ID                int64           `db:"id"`
	UID               string          `db:"uid"`
	RequestID         *int64          `db:"request_id"`
	RequestUID        string          `db:"request_uid"`
	DeliveryAttemptID *int64          `db:"delivery_attempt_id"`
	DeliveryUID       string          `db:"delivery_uid"`
	AsyncTaskID       *int64          `db:"async_task_id"`
	AsyncTaskUID      string          `db:"async_task_uid"`
	WorkerRunID       *int64          `db:"worker_run_id"`
	WorkerRunUID      string          `db:"worker_run_uid"`
	EventType         string          `db:"event_type"`
	EventLevel        string          `db:"event_level"`
	EventData         json.RawMessage `db:"event_data"`
	Message           string          `db:"message"`
	CorrelationID     string          `db:"correlation_id"`
	ActorType         string          `db:"actor_type"`
	ActorUserID       *int64          `db:"actor_user_id"`
	ActorName         string          `db:"actor_name"`
	SourceComponent   string          `db:"source_component"`
	CreatedAt         time.Time       `db:"created_at"`
}

func (r *Repository) AppendEvent(ctx context.Context, input EventWriteInput) (EventRecord, error) {
	if r.db == nil {
		return EventRecord{}, errors.New("observability repository requires database")
	}
	payload, err := json.Marshal(sanitizeEventData(input.EventData))
	if err != nil {
		return EventRecord{}, fmt.Errorf("marshal request event data: %w", err)
	}

	var id int64
	if err := r.db.GetContext(ctx, &id, `
		INSERT INTO request_events (
			uid, request_id, delivery_attempt_id, async_task_id, worker_run_id,
			event_type, event_level, event_data, message, correlation_id,
			actor_type, actor_user_id, actor_name, source_component, created_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, NULLIF($9, ''), NULLIF($10, ''), $11, $12, NULLIF($13, ''), NULLIF($14, ''), NOW())
		RETURNING id
	`,
		newUID(),
		input.RequestID,
		input.DeliveryAttemptID,
		input.AsyncTaskID,
		input.WorkerRunID,
		input.EventType,
		normalizeLevel(input.EventLevel),
		string(payload),
		input.Message,
		input.CorrelationID,
		normalizeActorType(input.Actor.Type),
		input.Actor.UserID,
		input.Actor.Name,
		input.SourceComponent,
	); err != nil {
		return EventRecord{}, fmt.Errorf("insert request event: %w", err)
	}
	return r.GetEvent(ctx, id)
}

func (r *Repository) GetEvent(ctx context.Context, id int64) (EventRecord, error) {
	if r.db == nil {
		return EventRecord{}, errors.New("observability repository requires database")
	}
	var row eventRow
	if err := r.db.GetContext(ctx, &row, baseEventSelect()+` WHERE e.id = $1`, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return EventRecord{}, sql.ErrNoRows
		}
		return EventRecord{}, fmt.Errorf("get request event: %w", err)
	}
	return decodeEventRow(row, true)
}

func (r *Repository) ListEvents(ctx context.Context, query EventListQuery) (EventListResult, error) {
	if r.db == nil {
		return EventListResult{}, errors.New("observability repository requires database")
	}
	q := normalizeEventListQuery(query)
	conditions := make([]string, 0, 8)
	args := make([]any, 0, 12)
	if q.RequestID != nil {
		args = append(args, *q.RequestID)
		conditions = append(conditions, fmt.Sprintf("e.request_id = $%d", len(args)))
	}
	if q.DeliveryAttemptID != nil {
		args = append(args, *q.DeliveryAttemptID)
		conditions = append(conditions, fmt.Sprintf("e.delivery_attempt_id = $%d", len(args)))
	}
	if q.AsyncTaskID != nil {
		args = append(args, *q.AsyncTaskID)
		conditions = append(conditions, fmt.Sprintf("e.async_task_id = $%d", len(args)))
	}
	if q.WorkerRunID != nil {
		args = append(args, *q.WorkerRunID)
		conditions = append(conditions, fmt.Sprintf("e.worker_run_id = $%d", len(args)))
	}
	if q.CorrelationID != "" {
		args = append(args, q.CorrelationID)
		conditions = append(conditions, fmt.Sprintf("e.correlation_id = $%d", len(args)))
	}
	if q.EventType != "" {
		args = append(args, q.EventType)
		conditions = append(conditions, fmt.Sprintf("e.event_type = $%d", len(args)))
	}
	if q.Level != "" {
		args = append(args, q.Level)
		conditions = append(conditions, fmt.Sprintf("e.event_level = $%d", len(args)))
	}
	if q.From != nil {
		args = append(args, *q.From)
		conditions = append(conditions, fmt.Sprintf("e.created_at >= $%d", len(args)))
	}
	if q.To != nil {
		args = append(args, *q.To)
		conditions = append(conditions, fmt.Sprintf("e.created_at <= $%d", len(args)))
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = " WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM request_events e`+whereClause, args...); err != nil {
		return EventListResult{}, fmt.Errorf("count request events: %w", err)
	}

	selectArgs := append([]any{}, args...)
	offset := (q.Page - 1) * q.PageSize
	selectArgs = append(selectArgs, q.PageSize, offset)
	rows := []eventRow{}
	if err := r.db.SelectContext(
		ctx,
		&rows,
		baseEventSelect()+whereClause+fmt.Sprintf(" ORDER BY e.created_at %s, e.id %s LIMIT $%d OFFSET $%d", strings.ToUpper(q.SortOrder), strings.ToUpper(q.SortOrder), len(selectArgs)-1, len(selectArgs)),
		selectArgs...,
	); err != nil {
		return EventListResult{}, fmt.Errorf("list request events: %w", err)
	}

	items := make([]EventRecord, 0, len(rows))
	for _, row := range rows {
		item, err := decodeEventRow(row, false)
		if err != nil {
			return EventListResult{}, err
		}
		items = append(items, item)
	}
	return EventListResult{Items: items, Total: total, Page: q.Page, PageSize: q.PageSize}, nil
}

func (r *Repository) TraceByCorrelationID(ctx context.Context, correlationID string) (TraceResult, error) {
	list, err := r.ListEvents(ctx, EventListQuery{
		Page:          1,
		PageSize:      200,
		CorrelationID: strings.TrimSpace(correlationID),
		SortOrder:     "asc",
	})
	if err != nil {
		return TraceResult{}, err
	}
	trace := TraceResult{
		CorrelationID: strings.TrimSpace(correlationID),
		Events:        list.Items,
		Summary: TraceSummary{
			Requests:   []TraceReference{},
			Deliveries: []TraceReference{},
			Jobs:       []TraceReference{},
			Workers:    []TraceReference{},
		},
	}
	reqSeen := map[int64]struct{}{}
	delSeen := map[int64]struct{}{}
	jobSeen := map[int64]struct{}{}
	workerSeen := map[int64]struct{}{}
	for _, item := range list.Items {
		if item.RequestID != nil {
			if _, ok := reqSeen[*item.RequestID]; !ok {
				reqSeen[*item.RequestID] = struct{}{}
				trace.Summary.Requests = append(trace.Summary.Requests, TraceReference{ID: *item.RequestID, UID: item.RequestUID, CreatedAt: item.CreatedAt})
			}
		}
		if item.DeliveryAttemptID != nil {
			if _, ok := delSeen[*item.DeliveryAttemptID]; !ok {
				delSeen[*item.DeliveryAttemptID] = struct{}{}
				trace.Summary.Deliveries = append(trace.Summary.Deliveries, TraceReference{ID: *item.DeliveryAttemptID, UID: item.DeliveryUID, CreatedAt: item.CreatedAt})
			}
		}
		if item.AsyncTaskID != nil {
			if _, ok := jobSeen[*item.AsyncTaskID]; !ok {
				jobSeen[*item.AsyncTaskID] = struct{}{}
				trace.Summary.Jobs = append(trace.Summary.Jobs, TraceReference{ID: *item.AsyncTaskID, UID: item.AsyncTaskUID, CreatedAt: item.CreatedAt})
			}
		}
		if item.WorkerRunID != nil {
			if _, ok := workerSeen[*item.WorkerRunID]; !ok {
				workerSeen[*item.WorkerRunID] = struct{}{}
				trace.Summary.Workers = append(trace.Summary.Workers, TraceReference{ID: *item.WorkerRunID, UID: item.WorkerRunUID, CreatedAt: item.CreatedAt})
			}
		}
	}
	return trace, nil
}

func (r *Repository) HasRequest(ctx context.Context, id int64) (bool, error) {
	return r.exists(ctx, `SELECT EXISTS(SELECT 1 FROM exchange_requests WHERE id = $1)`, id)
}

func (r *Repository) HasDelivery(ctx context.Context, id int64) (bool, error) {
	return r.exists(ctx, `SELECT EXISTS(SELECT 1 FROM delivery_attempts WHERE id = $1)`, id)
}

func (r *Repository) HasJob(ctx context.Context, id int64) (bool, error) {
	return r.exists(ctx, `SELECT EXISTS(SELECT 1 FROM async_tasks WHERE id = $1)`, id)
}

func (r *Repository) exists(ctx context.Context, sqlText string, id int64) (bool, error) {
	if r.db == nil {
		return false, errors.New("observability repository requires database")
	}
	var exists bool
	if err := r.db.GetContext(ctx, &exists, sqlText, id); err != nil {
		return false, err
	}
	return exists, nil
}

func baseEventSelect() string {
	return `
		SELECT e.id, e.uid::text AS uid, e.request_id, COALESCE(r.uid::text, '') AS request_uid,
		       e.delivery_attempt_id, COALESCE(d.uid::text, '') AS delivery_uid,
		       e.async_task_id, COALESCE(a.uid::text, '') AS async_task_uid,
		       e.worker_run_id, COALESCE(w.uid::text, '') AS worker_run_uid,
		       e.event_type, e.event_level, e.event_data,
		       COALESCE(e.message, '') AS message, COALESCE(e.correlation_id, '') AS correlation_id,
		       e.actor_type, e.actor_user_id, COALESCE(e.actor_name, '') AS actor_name,
		       COALESCE(e.source_component, '') AS source_component, e.created_at
		FROM request_events e
		LEFT JOIN exchange_requests r ON r.id = e.request_id
		LEFT JOIN delivery_attempts d ON d.id = e.delivery_attempt_id
		LEFT JOIN async_tasks a ON a.id = e.async_task_id
		LEFT JOIN worker_runs w ON w.id = e.worker_run_id
	`
}

func normalizeEventListQuery(query EventListQuery) EventListQuery {
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 || pageSize > 200 {
		pageSize = 25
	}
	sortOrder := strings.ToLower(strings.TrimSpace(query.SortOrder))
	if sortOrder != "asc" {
		sortOrder = "desc"
	}
	return EventListQuery{
		Page:              page,
		PageSize:          pageSize,
		RequestID:         query.RequestID,
		DeliveryAttemptID: query.DeliveryAttemptID,
		AsyncTaskID:       query.AsyncTaskID,
		WorkerRunID:       query.WorkerRunID,
		CorrelationID:     strings.TrimSpace(query.CorrelationID),
		EventType:         strings.TrimSpace(query.EventType),
		Level:             normalizeLevel(query.Level),
		From:              query.From,
		To:                query.To,
		SortOrder:         sortOrder,
	}
}

func decodeEventRow(row eventRow, includeData bool) (EventRecord, error) {
	data := map[string]any{}
	if len(row.EventData) > 0 {
		if err := json.Unmarshal(row.EventData, &data); err != nil {
			return EventRecord{}, fmt.Errorf("decode request event data: %w", err)
		}
	}
	record := EventRecord{
		ID:                row.ID,
		UID:               row.UID,
		RequestID:         row.RequestID,
		RequestUID:        row.RequestUID,
		DeliveryAttemptID: row.DeliveryAttemptID,
		DeliveryUID:       row.DeliveryUID,
		AsyncTaskID:       row.AsyncTaskID,
		AsyncTaskUID:      row.AsyncTaskUID,
		WorkerRunID:       row.WorkerRunID,
		WorkerRunUID:      row.WorkerRunUID,
		EventType:         row.EventType,
		EventLevel:        row.EventLevel,
		EventDataPreview:  previewEventData(data),
		Message:           row.Message,
		CorrelationID:     row.CorrelationID,
		ActorType:         row.ActorType,
		ActorUserID:       row.ActorUserID,
		ActorName:         row.ActorName,
		SourceComponent:   row.SourceComponent,
		CreatedAt:         row.CreatedAt,
	}
	if includeData {
		record.EventData = data
	}
	return record, nil
}

func newUID() string {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		now := time.Now().UTC()
		return fmt.Sprintf("%d", now.UnixNano())
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(buf)
	return fmt.Sprintf("%s-%s-%s-%s-%s", encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32])
}
