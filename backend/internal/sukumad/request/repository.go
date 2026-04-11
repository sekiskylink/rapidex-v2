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
	ID                      int64           `db:"id"`
	UID                     string          `db:"uid"`
	SourceSystem            string          `db:"source_system"`
	DestinationServerID     int64           `db:"destination_server_id"`
	DestinationServerUID    string          `db:"destination_server_uid"`
	DestinationServerName   string          `db:"destination_server_name"`
	DestinationServerCode   string          `db:"destination_server_code"`
	BatchID                 string          `db:"batch_id"`
	CorrelationID           string          `db:"correlation_id"`
	IdempotencyKey          string          `db:"idempotency_key"`
	PayloadBody             string          `db:"payload_body"`
	PayloadFormat           string          `db:"payload_format"`
	SubmissionBinding       string          `db:"submission_binding"`
	ResponseBodyPersistence string          `db:"response_body_persistence"`
	URLSuffix               string          `db:"url_suffix"`
	Status                  string          `db:"status"`
	StatusReason            string          `db:"status_reason"`
	DeferredUntil           *time.Time      `db:"deferred_until"`
	Extras                  json.RawMessage `db:"extras"`
	CreatedAt               time.Time       `db:"created_at"`
	UpdatedAt               time.Time       `db:"updated_at"`
	CreatedBy               *int64          `db:"created_by"`
	LatestDeliveryID        *int64          `db:"latest_delivery_id"`
	LatestDeliveryUID       string          `db:"latest_delivery_uid"`
	LatestDeliveryStatus    string          `db:"latest_delivery_status"`
	LatestAsyncTaskID       *int64          `db:"latest_async_task_id"`
	LatestAsyncTaskUID      string          `db:"latest_async_task_uid"`
	LatestAsyncState        string          `db:"latest_async_state"`
	LatestAsyncRemoteJobID  string          `db:"latest_async_remote_job_id"`
	LatestAsyncPollURL      string          `db:"latest_async_poll_url"`
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
		Page:            page,
		PageSize:        pageSize,
		SortField:       sortField,
		SortOrder:       sortOrder,
		Filter:          strings.TrimSpace(query.Filter),
		Status:          strings.ToLower(strings.TrimSpace(query.Status)),
		MetadataColumns: normalizeMetadataColumns(query.MetadataColumns),
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
		filterClauses := []string{
			`(
			r.uid::text ILIKE ` + needle + ` OR
			COALESCE(r.source_system, '') ILIKE ` + needle + ` OR
			COALESCE(r.correlation_id, '') ILIKE ` + needle + ` OR
			COALESCE(r.batch_id, '') ILIKE ` + needle + ` OR
			COALESCE(r.idempotency_key, '') ILIKE ` + needle + ` OR
			COALESCE(r.url_suffix, '') ILIKE ` + needle + ` OR
			COALESCE(s.name, '') ILIKE ` + needle + ` OR
			COALESCE(s.code, '') ILIKE ` + needle + `
		)`,
		}
		for _, column := range searchableMetadataColumns(q.MetadataColumns) {
			filterClauses = append(filterClauses, fmt.Sprintf("COALESCE(r.extras ->> '%s', '') ILIKE %s", escapeSQLLiteral(column.Key), needle))
		}
		conditions = append(conditions, "("+strings.Join(filterClauses, " OR ")+")")
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
		       COALESCE(s.uid::text, '') AS destination_server_uid,
		       COALESCE(s.name, '') AS destination_server_name,
		       COALESCE(s.code, '') AS destination_server_code,
		       COALESCE(r.batch_id, '') AS batch_id, COALESCE(r.correlation_id, '') AS correlation_id, COALESCE(r.idempotency_key, '') AS idempotency_key,
		       r.payload_body, r.payload_format, r.submission_binding, COALESCE(r.response_body_persistence, '') AS response_body_persistence, COALESCE(r.url_suffix, '') AS url_suffix, r.status, COALESCE(r.status_reason, '') AS status_reason, r.deferred_until,
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
	if err := r.hydrateRequests(ctx, items); err != nil {
		return ListResult{}, err
	}

	return ListResult{
		Items:           items,
		Total:           total,
		Page:            q.Page,
		PageSize:        q.PageSize,
		MetadataColumns: q.MetadataColumns,
	}, nil
}

func escapeSQLLiteral(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func (r *SQLRepository) GetRequestByID(ctx context.Context, id int64) (Record, error) {
	return r.getRequestByWhere(ctx, "r.id = $1", id)
}

func (r *SQLRepository) GetRequestByUID(ctx context.Context, uid string) (Record, error) {
	return r.getRequestByWhere(ctx, "r.uid::text = $1", strings.TrimSpace(uid))
}

func (r *SQLRepository) ListRequestsByBatchID(ctx context.Context, batchID string) ([]Record, error) {
	return r.listRequestsByWhere(ctx, "r.batch_id = $1", strings.TrimSpace(batchID))
}

func (r *SQLRepository) ListRequestsByCorrelationID(ctx context.Context, correlationID string) ([]Record, error) {
	return r.listRequestsByWhere(ctx, "r.correlation_id = $1", strings.TrimSpace(correlationID))
}

func (r *SQLRepository) GetRequestBySourceSystemAndIdempotencyKey(ctx context.Context, sourceSystem string, idempotencyKey string) (Record, error) {
	return r.getRequestByWhere(ctx, "r.source_system = $1 AND r.idempotency_key = $2", strings.TrimSpace(sourceSystem), strings.TrimSpace(idempotencyKey))
}

func (r *SQLRepository) getRequestByWhere(ctx context.Context, whereClause string, args ...any) (Record, error) {
	var row recordRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT r.id, r.uid::text AS uid, r.source_system, r.destination_server_id,
		       COALESCE(s.uid::text, '') AS destination_server_uid,
		       COALESCE(s.name, '') AS destination_server_name,
		       COALESCE(s.code, '') AS destination_server_code,
		       COALESCE(r.batch_id, '') AS batch_id, COALESCE(r.correlation_id, '') AS correlation_id, COALESCE(r.idempotency_key, '') AS idempotency_key,
		       r.payload_body, r.payload_format, r.submission_binding, COALESCE(r.response_body_persistence, '') AS response_body_persistence, COALESCE(r.url_suffix, '') AS url_suffix, r.status, COALESCE(r.status_reason, '') AS status_reason, r.deferred_until,
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
		WHERE `+whereClause+`
	`, args...); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, sql.ErrNoRows
		}
		return Record{}, fmt.Errorf("get exchange request: %w", err)
	}
	record, err := decodeRow(row)
	if err != nil {
		return Record{}, err
	}
	items := []Record{record}
	if err := r.hydrateRequests(ctx, items); err != nil {
		return Record{}, err
	}
	return items[0], nil
}

func (r *SQLRepository) listRequestsByWhere(ctx context.Context, whereClause string, args ...any) ([]Record, error) {
	rows := []recordRow{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT r.id, r.uid::text AS uid, r.source_system, r.destination_server_id,
		       COALESCE(s.uid::text, '') AS destination_server_uid,
		       COALESCE(s.name, '') AS destination_server_name,
		       COALESCE(s.code, '') AS destination_server_code,
		       COALESCE(r.batch_id, '') AS batch_id, COALESCE(r.correlation_id, '') AS correlation_id, COALESCE(r.idempotency_key, '') AS idempotency_key,
		       r.payload_body, r.payload_format, r.submission_binding, COALESCE(r.response_body_persistence, '') AS response_body_persistence, COALESCE(r.url_suffix, '') AS url_suffix, r.status, COALESCE(r.status_reason, '') AS status_reason, r.deferred_until,
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
		WHERE `+whereClause+`
		ORDER BY r.created_at DESC, r.id DESC
	`, args...); err != nil {
		return nil, fmt.Errorf("list exchange requests by selector: %w", err)
	}
	items, err := decodeRows(rows)
	if err != nil {
		return nil, err
	}
	if err := r.hydrateRequests(ctx, items); err != nil {
		return nil, err
	}
	return items, nil
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
			payload_body, payload_format, submission_binding, response_body_persistence, url_suffix, status, status_reason, deferred_until, extras, created_at, updated_at, created_by
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15::jsonb, NOW(), NOW(), $16)
		RETURNING id
	`,
		params.UID,
		params.SourceSystem,
		params.DestinationServerID,
		params.BatchID,
		params.CorrelationID,
		params.IdempotencyKey,
		params.PayloadBody,
		params.PayloadFormat,
		params.SubmissionBinding,
		params.ResponseBodyPersistence,
		nullIfEmpty(params.URLSuffix),
		params.Status,
		params.StatusReason,
		params.DeferredUntil,
		string(extras),
		params.CreatedBy,
	); err != nil {
		return Record{}, fmt.Errorf("create exchange request: %w", err)
	}

	return r.GetRequestByID(ctx, id)
}

func (r *SQLRepository) UpdateRequestStatus(ctx context.Context, id int64, status string, reason string, deferredUntil *time.Time) (Record, error) {
	var updatedID int64
	if err := r.db.GetContext(ctx, &updatedID, `
		UPDATE exchange_requests
		SET status = $2,
		    status_reason = $3,
		    deferred_until = $4,
		    updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, id, status, reason, deferredUntil); err != nil {
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
		ID:                      row.ID,
		UID:                     row.UID,
		SourceSystem:            row.SourceSystem,
		DestinationServerID:     row.DestinationServerID,
		DestinationServerUID:    row.DestinationServerUID,
		DestinationServerName:   row.DestinationServerName,
		DestinationServerCode:   row.DestinationServerCode,
		BatchID:                 row.BatchID,
		CorrelationID:           row.CorrelationID,
		IdempotencyKey:          row.IdempotencyKey,
		PayloadBody:             row.PayloadBody,
		PayloadFormat:           row.PayloadFormat,
		SubmissionBinding:       row.SubmissionBinding,
		ResponseBodyPersistence: row.ResponseBodyPersistence,
		URLSuffix:               row.URLSuffix,
		Status:                  row.Status,
		StatusReason:            row.StatusReason,
		DeferredUntil:           cloneTimePtr(row.DeferredUntil),
		Extras:                  extras,
		CreatedAt:               row.CreatedAt,
		UpdatedAt:               row.UpdatedAt,
		CreatedBy:               row.CreatedBy,
		Payload:                 decodePayload(row.PayloadBody, row.PayloadFormat),
		LatestDeliveryID:        cloneInt64Ptr(row.LatestDeliveryID),
		LatestDeliveryUID:       row.LatestDeliveryUID,
		LatestDeliveryStatus:    row.LatestDeliveryStatus,
		LatestAsyncTaskID:       cloneInt64Ptr(row.LatestAsyncTaskID),
		LatestAsyncTaskUID:      row.LatestAsyncTaskUID,
		LatestAsyncState:        row.LatestAsyncState,
		LatestAsyncRemoteJobID:  row.LatestAsyncRemoteJobID,
		LatestAsyncPollURL:      row.LatestAsyncPollURL,
		AwaitingAsync:           row.LatestAsyncTaskID != nil && row.LatestAsyncState != "" && row.LatestAsyncState != StatusCompleted && row.LatestAsyncState != StatusFailed,
	}, nil
}

func (r *SQLRepository) DeleteRequest(ctx context.Context, id int64) error {
	tx, err := r.db.BeginTxx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin delete exchange request transaction: %w", err)
	}
	defer func() {
		_ = tx.Rollback()
	}()

	steps := []string{
		`
			WITH target_async_tasks AS (
				SELECT a.id
				FROM async_tasks a
				JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
				WHERE d.request_id = $1
			)
			DELETE FROM async_task_polls
			WHERE async_task_id IN (SELECT id FROM target_async_tasks)
		`,
		`
			DELETE FROM async_tasks
			WHERE delivery_attempt_id IN (
				SELECT id FROM delivery_attempts WHERE request_id = $1
			)
		`,
		`DELETE FROM request_events WHERE request_id = $1`,
		`DELETE FROM delivery_attempts WHERE request_id = $1`,
		`DELETE FROM request_targets WHERE request_id = $1`,
		`DELETE FROM request_dependencies WHERE request_id = $1 OR depends_on_request_id = $1`,
	}

	for _, query := range steps {
		if _, err := tx.ExecContext(ctx, query, id); err != nil {
			return fmt.Errorf("delete exchange request dependencies: %w", err)
		}
	}

	result, err := tx.ExecContext(ctx, `DELETE FROM exchange_requests WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete exchange request: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete exchange request rows affected: %w", err)
	}
	if affected == 0 {
		return sql.ErrNoRows
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete exchange request transaction: %w", err)
	}
	return nil
}

func (r *SQLRepository) hydrateRequests(ctx context.Context, items []Record) error {
	if len(items) == 0 {
		return nil
	}

	requestIDs := make([]int64, 0, len(items))
	indexByID := make(map[int64]int, len(items))
	for index := range items {
		requestIDs = append(requestIDs, items[index].ID)
		indexByID[items[index].ID] = index
	}

	targetsByRequestID, err := r.listTargetsByRequestIDs(ctx, requestIDs)
	if err != nil {
		return err
	}
	dependenciesByRequestID, err := r.listDependenciesByRequestIDs(ctx, requestIDs)
	if err != nil {
		return err
	}

	for requestID, targets := range targetsByRequestID {
		items[indexByID[requestID]].Targets = targets
	}
	for requestID, dependencies := range dependenciesByRequestID {
		items[indexByID[requestID]].Dependencies = dependencies
	}
	return nil
}

func (r *SQLRepository) listTargetsByRequestIDs(ctx context.Context, requestIDs []int64) (map[int64][]TargetRecord, error) {
	query, args, err := sqlx.In(`
		SELECT t.id, t.uid::text AS uid, t.request_id, t.server_id, COALESCE(s.uid::text, '') AS server_uid, COALESCE(s.name, '') AS server_name, COALESCE(s.code, '') AS server_code,
		       t.target_kind, t.priority, t.status, COALESCE(t.blocked_reason, '') AS blocked_reason, t.deferred_until, t.last_released_at,
		       ld.id AS latest_delivery_id,
		       COALESCE(ld.uid, '') AS latest_delivery_uid,
		       COALESCE(ld.status, '') AS latest_delivery_status,
		       a.id AS latest_async_task_id,
		       COALESCE(a.uid::text, '') AS latest_async_task_uid,
		       COALESCE(a.terminal_state, CASE WHEN COALESCE(a.remote_status, '') = '' THEN '' ELSE a.remote_status END) AS latest_async_state,
		       COALESCE(a.remote_job_id, '') AS latest_async_remote_job_id,
		       COALESCE(a.poll_url, '') AS latest_async_poll_url,
		       t.created_at, t.updated_at
		FROM request_targets t
		LEFT JOIN integration_servers s ON s.id = t.server_id
		LEFT JOIN LATERAL (
			SELECT d.id,
			       d.uid::text AS uid,
			       d.status
			FROM delivery_attempts d
			WHERE d.request_id = t.request_id
			  AND d.server_id = t.server_id
			ORDER BY d.attempt_number DESC, d.created_at DESC
			LIMIT 1
		) ld ON TRUE
		LEFT JOIN async_tasks a ON a.delivery_attempt_id = ld.id
		WHERE t.request_id IN (?)
		ORDER BY t.request_id ASC, t.priority ASC, t.id ASC
	`, requestIDs)
	if err != nil {
		return nil, fmt.Errorf("build request target hydration query: %w", err)
	}
	query = r.db.Rebind(query)

	type targetRow struct {
		ID                     int64      `db:"id"`
		UID                    string     `db:"uid"`
		RequestID              int64      `db:"request_id"`
		ServerID               int64      `db:"server_id"`
		ServerUID              string     `db:"server_uid"`
		ServerName             string     `db:"server_name"`
		ServerCode             string     `db:"server_code"`
		TargetKind             string     `db:"target_kind"`
		Priority               int        `db:"priority"`
		Status                 string     `db:"status"`
		BlockedReason          string     `db:"blocked_reason"`
		DeferredUntil          *time.Time `db:"deferred_until"`
		LastReleasedAt         *time.Time `db:"last_released_at"`
		LatestDeliveryID       *int64     `db:"latest_delivery_id"`
		LatestDeliveryUID      string     `db:"latest_delivery_uid"`
		LatestDeliveryStatus   string     `db:"latest_delivery_status"`
		LatestAsyncTaskID      *int64     `db:"latest_async_task_id"`
		LatestAsyncTaskUID     string     `db:"latest_async_task_uid"`
		LatestAsyncState       string     `db:"latest_async_state"`
		LatestAsyncRemoteJobID string     `db:"latest_async_remote_job_id"`
		LatestAsyncPollURL     string     `db:"latest_async_poll_url"`
		CreatedAt              time.Time  `db:"created_at"`
		UpdatedAt              time.Time  `db:"updated_at"`
	}

	rows := []targetRow{}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("hydrate request targets: %w", err)
	}

	result := make(map[int64][]TargetRecord, len(requestIDs))
	for _, row := range rows {
		target := TargetRecord{
			ID:                     row.ID,
			UID:                    row.UID,
			RequestID:              row.RequestID,
			ServerID:               row.ServerID,
			ServerUID:              row.ServerUID,
			ServerName:             row.ServerName,
			ServerCode:             row.ServerCode,
			TargetKind:             row.TargetKind,
			Priority:               row.Priority,
			Status:                 row.Status,
			BlockedReason:          row.BlockedReason,
			DeferredUntil:          cloneTimePtr(row.DeferredUntil),
			LastReleasedAt:         cloneTimePtr(row.LastReleasedAt),
			LatestDeliveryID:       cloneInt64Ptr(row.LatestDeliveryID),
			LatestDeliveryUID:      row.LatestDeliveryUID,
			LatestDeliveryStatus:   row.LatestDeliveryStatus,
			LatestAsyncTaskID:      cloneInt64Ptr(row.LatestAsyncTaskID),
			LatestAsyncTaskUID:     row.LatestAsyncTaskUID,
			LatestAsyncState:       row.LatestAsyncState,
			LatestAsyncRemoteJobID: row.LatestAsyncRemoteJobID,
			LatestAsyncPollURL:     row.LatestAsyncPollURL,
			AwaitingAsync:          row.LatestAsyncTaskID != nil && row.LatestAsyncState != "" && row.LatestAsyncState != StatusCompleted && row.LatestAsyncState != StatusFailed,
			CreatedAt:              row.CreatedAt,
			UpdatedAt:              row.UpdatedAt,
		}
		result[row.RequestID] = append(result[row.RequestID], target)
	}

	return result, nil
}

func (r *SQLRepository) listDependenciesByRequestIDs(ctx context.Context, requestIDs []int64) (map[int64][]DependencyRef, error) {
	query, args, err := sqlx.In(`
		SELECT d.request_id, d.depends_on_request_id, COALESCE(r.uid::text, '') AS request_uid,
		       COALESCE(dep.uid::text, '') AS depends_on_uid, COALESCE(dep.status, '') AS status,
		       COALESCE(dep.status_reason, '') AS status_reason, dep.deferred_until,
		       COALESCE(s.name, '') AS depends_on_destination_server_name
		FROM request_dependencies d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN exchange_requests dep ON dep.id = d.depends_on_request_id
		LEFT JOIN integration_servers s ON s.id = dep.destination_server_id
		WHERE d.request_id IN (?)
		ORDER BY d.request_id ASC, d.depends_on_request_id ASC
	`, requestIDs)
	if err != nil {
		return nil, fmt.Errorf("build request dependency hydration query: %w", err)
	}
	query = r.db.Rebind(query)

	rows := []DependencyRef{}
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, fmt.Errorf("hydrate request dependencies: %w", err)
	}

	result := make(map[int64][]DependencyRef, len(requestIDs))
	for _, row := range rows {
		row.DeferredUntil = cloneTimePtr(row.DeferredUntil)
		result[row.RequestID] = append(result[row.RequestID], row)
	}
	return result, nil
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

func decodePayload(body string, payloadFormat string) any {
	if payloadFormat == PayloadFormatText {
		return body
	}
	var parsed any
	if err := json.Unmarshal([]byte(body), &parsed); err != nil {
		return body
	}
	return parsed
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

func (r *memoryRepository) GetRequestByUID(_ context.Context, uid string) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, item := range r.items {
		if item.UID == strings.TrimSpace(uid) {
			return cloneRecord(item), nil
		}
	}
	return Record{}, sql.ErrNoRows
}

func (r *memoryRepository) ListRequestsByBatchID(_ context.Context, batchID string) ([]Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := []Record{}
	for _, item := range r.items {
		if item.BatchID == strings.TrimSpace(batchID) {
			items = append(items, cloneRecord(item))
		}
	}
	slices.SortFunc(items, func(a, b Record) int { return compareTimes(b.CreatedAt, a.CreatedAt) })
	return items, nil
}

func (r *memoryRepository) ListRequestsByCorrelationID(_ context.Context, correlationID string) ([]Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := []Record{}
	for _, item := range r.items {
		if item.CorrelationID == strings.TrimSpace(correlationID) {
			items = append(items, cloneRecord(item))
		}
	}
	slices.SortFunc(items, func(a, b Record) int { return compareTimes(b.CreatedAt, a.CreatedAt) })
	return items, nil
}

func (r *memoryRepository) GetRequestBySourceSystemAndIdempotencyKey(_ context.Context, sourceSystem string, idempotencyKey string) (Record, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	sourceSystem = strings.TrimSpace(sourceSystem)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	for _, item := range r.items {
		if item.SourceSystem == sourceSystem && item.IdempotencyKey == idempotencyKey {
			return cloneRecord(item), nil
		}
	}
	return Record{}, sql.ErrNoRows
}

func (r *memoryRepository) CreateRequest(_ context.Context, params CreateParams) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, existing := range r.items {
		if existing.SourceSystem == params.SourceSystem && existing.IdempotencyKey != "" && existing.IdempotencyKey == params.IdempotencyKey {
			return Record{}, fmt.Errorf("create exchange request: duplicate idempotency key")
		}
	}

	id := r.nextID
	r.nextID++
	now := time.Now().UTC()
	record := Record{
		ID:                      id,
		UID:                     params.UID,
		SourceSystem:            params.SourceSystem,
		DestinationServerID:     params.DestinationServerID,
		DestinationServerUID:    fmt.Sprintf("server-uid-%d", params.DestinationServerID),
		DestinationServerName:   fmt.Sprintf("Server #%d", params.DestinationServerID),
		DestinationServerCode:   fmt.Sprintf("server-%d", params.DestinationServerID),
		BatchID:                 params.BatchID,
		CorrelationID:           params.CorrelationID,
		IdempotencyKey:          params.IdempotencyKey,
		PayloadBody:             params.PayloadBody,
		PayloadFormat:           params.PayloadFormat,
		SubmissionBinding:       params.SubmissionBinding,
		ResponseBodyPersistence: params.ResponseBodyPersistence,
		URLSuffix:               params.URLSuffix,
		Status:                  params.Status,
		StatusReason:            params.StatusReason,
		DeferredUntil:           cloneTimePtr(params.DeferredUntil),
		Extras:                  cloneExtras(params.Extras),
		CreatedAt:               now,
		UpdatedAt:               now,
		CreatedBy:               params.CreatedBy,
		Payload:                 decodePayload(params.PayloadBody, params.PayloadFormat),
	}
	r.items[id] = record
	return cloneRecord(record), nil
}

func (r *memoryRepository) DeleteRequest(_ context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[id]; !ok {
		return sql.ErrNoRows
	}
	delete(r.items, id)
	for itemID, item := range r.items {
		filtered := item.Dependencies[:0]
		for _, dependency := range item.Dependencies {
			if dependency.RequestID != id && dependency.DependsOnRequestID != id {
				filtered = append(filtered, dependency)
			}
		}
		item.Dependencies = filtered
		r.items[itemID] = item
	}
	return nil
}

func (r *memoryRepository) UpdateRequestStatus(_ context.Context, id int64, status string, reason string, deferredUntil *time.Time) (Record, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[id]
	if !ok {
		return Record{}, sql.ErrNoRows
	}
	item.Status = status
	item.StatusReason = reason
	item.DeferredUntil = cloneTimePtr(deferredUntil)
	item.UpdatedAt = time.Now().UTC()
	r.items[id] = item
	return cloneRecord(item), nil
}

func (r *SQLRepository) CreateTargets(ctx context.Context, requestID int64, targets []CreateTargetParams) ([]TargetRecord, error) {
	items := make([]TargetRecord, 0, len(targets))
	for _, target := range targets {
		var item TargetRecord
		if err := r.db.GetContext(ctx, &item, `
			INSERT INTO request_targets (uid, request_id, server_id, target_kind, priority, status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			RETURNING id, uid::text AS uid, request_id, server_id, target_kind, priority, status, blocked_reason, deferred_until, last_released_at, created_at, updated_at
		`, target.UID, requestID, target.ServerID, target.TargetKind, target.Priority, target.Status); err != nil {
			return nil, fmt.Errorf("create request target: %w", err)
		}
		items = append(items, item)
	}
	return r.ListTargetsByRequest(ctx, requestID)
}

func (r *SQLRepository) ListTargetsByRequest(ctx context.Context, requestID int64) ([]TargetRecord, error) {
	rowsByRequestID, err := r.listTargetsByRequestIDs(ctx, []int64{requestID})
	if err != nil {
		return nil, fmt.Errorf("list request targets: %w", err)
	}
	rows, ok := rowsByRequestID[requestID]
	if !ok {
		return []TargetRecord{}, nil
	}
	return rows, nil
}

func (r *SQLRepository) UpdateTarget(ctx context.Context, params UpdateTargetParams) (TargetRecord, error) {
	var touched int
	if err := r.db.GetContext(ctx, &touched, `
		UPDATE request_targets
		SET status = $3,
		    blocked_reason = $4,
		    deferred_until = $5,
		    last_released_at = $6,
		    updated_at = NOW()
		WHERE request_id = $1
		  AND server_id = $2
		RETURNING 1
	`, params.RequestID, params.ServerID, params.Status, params.BlockedReason, params.DeferredUntil, params.LastReleasedAt); err != nil {
		return TargetRecord{}, fmt.Errorf("update request target: %w", err)
	}
	rowsByRequestID, err := r.listTargetsByRequestIDs(ctx, []int64{params.RequestID})
	if err != nil {
		return TargetRecord{}, fmt.Errorf("list request targets: %w", err)
	}
	for _, item := range rowsByRequestID[params.RequestID] {
		if item.ServerID == params.ServerID {
			return item, nil
		}
	}
	return TargetRecord{}, sql.ErrNoRows
}

func (r *SQLRepository) CreateDependencies(ctx context.Context, requestID int64, dependencyIDs []int64) error {
	for _, dependencyID := range dependencyIDs {
		if _, err := r.db.ExecContext(ctx, `
			INSERT INTO request_dependencies (request_id, depends_on_request_id, created_at)
			VALUES ($1, $2, NOW())
		`, requestID, dependencyID); err != nil {
			return fmt.Errorf("create request dependency: %w", err)
		}
	}
	return nil
}

func (r *SQLRepository) ListDependencies(ctx context.Context, requestID int64) ([]DependencyRef, error) {
	rows := []DependencyRef{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT d.request_id, d.depends_on_request_id, COALESCE(r.uid::text, '') AS request_uid,
		       COALESCE(dep.uid::text, '') AS depends_on_uid, COALESCE(dep.status, '') AS status
		FROM request_dependencies d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN exchange_requests dep ON dep.id = d.depends_on_request_id
		WHERE d.request_id = $1
		ORDER BY d.depends_on_request_id ASC
	`, requestID); err != nil {
		return nil, fmt.Errorf("list request dependencies: %w", err)
	}
	return rows, nil
}

func (r *SQLRepository) ListDependents(ctx context.Context, dependencyRequestID int64) ([]DependencyRef, error) {
	rows := []DependencyRef{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT d.request_id, d.depends_on_request_id, COALESCE(r.uid::text, '') AS request_uid,
		       COALESCE(dep.uid::text, '') AS depends_on_uid, COALESCE(r.status, '') AS status
		FROM request_dependencies d
		LEFT JOIN exchange_requests r ON r.id = d.request_id
		LEFT JOIN exchange_requests dep ON dep.id = d.depends_on_request_id
		WHERE d.depends_on_request_id = $1
		ORDER BY d.request_id ASC
	`, dependencyRequestID); err != nil {
		return nil, fmt.Errorf("list dependent requests: %w", err)
	}
	return rows, nil
}

func (r *SQLRepository) GetDependencyStatuses(ctx context.Context, requestID int64) ([]DependencyStatus, error) {
	rows := []DependencyStatus{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT dep.id AS request_id, dep.uid::text AS request_uid, dep.status, COALESCE(dep.status_reason, '') AS status_reason
		FROM request_dependencies d
		JOIN exchange_requests dep ON dep.id = d.depends_on_request_id
		WHERE d.request_id = $1
		ORDER BY dep.id ASC
	`, requestID); err != nil {
		return nil, fmt.Errorf("get dependency statuses: %w", err)
	}
	return rows, nil
}

func (r *SQLRepository) DependencyPathExists(ctx context.Context, fromRequestID int64, toRequestID int64) (bool, error) {
	var exists bool
	if err := r.db.GetContext(ctx, &exists, `
		WITH RECURSIVE dependency_chain AS (
			SELECT depends_on_request_id
			FROM request_dependencies
			WHERE request_id = $1
			UNION
			SELECT d.depends_on_request_id
			FROM request_dependencies d
			JOIN dependency_chain dc ON dc.depends_on_request_id = d.request_id
		)
		SELECT EXISTS(SELECT 1 FROM dependency_chain WHERE depends_on_request_id = $2)
	`, fromRequestID, toRequestID); err != nil {
		return false, fmt.Errorf("check dependency path: %w", err)
	}
	return exists, nil
}

func (r *memoryRepository) CreateTargets(_ context.Context, requestID int64, targets []CreateTargetParams) ([]TargetRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.items[requestID]
	now := time.Now().UTC()
	item.Targets = make([]TargetRecord, 0, len(targets))
	for index, target := range targets {
		item.Targets = append(item.Targets, TargetRecord{
			ID:         int64(index + 1),
			UID:        target.UID,
			RequestID:  requestID,
			ServerID:   target.ServerID,
			ServerUID:  fmt.Sprintf("server-uid-%d", target.ServerID),
			ServerName: fmt.Sprintf("Server #%d", target.ServerID),
			ServerCode: fmt.Sprintf("server-%d", target.ServerID),
			TargetKind: target.TargetKind,
			Priority:   target.Priority,
			Status:     target.Status,
			CreatedAt:  now,
			UpdatedAt:  now,
		})
	}
	r.items[requestID] = item
	return append([]TargetRecord{}, item.Targets...), nil
}

func (r *memoryRepository) ListTargetsByRequest(_ context.Context, requestID int64) ([]TargetRecord, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]TargetRecord{}, r.items[requestID].Targets...), nil
}

func (r *memoryRepository) UpdateTarget(_ context.Context, params UpdateTargetParams) (TargetRecord, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	item, ok := r.items[params.RequestID]
	if !ok {
		return TargetRecord{}, sql.ErrNoRows
	}
	for index := range item.Targets {
		if item.Targets[index].ServerID != params.ServerID {
			continue
		}
		item.Targets[index].Status = params.Status
		item.Targets[index].BlockedReason = params.BlockedReason
		item.Targets[index].DeferredUntil = cloneTimePtr(params.DeferredUntil)
		item.Targets[index].LastReleasedAt = cloneTimePtr(params.LastReleasedAt)
		item.Targets[index].UpdatedAt = time.Now().UTC()
		r.items[params.RequestID] = item
		return item.Targets[index], nil
	}
	return TargetRecord{}, sql.ErrNoRows
}

func (r *memoryRepository) CreateDependencies(_ context.Context, requestID int64, dependencyIDs []int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	item := r.items[requestID]
	item.Dependencies = make([]DependencyRef, 0, len(dependencyIDs))
	for _, dependencyID := range dependencyIDs {
		dependency := r.items[dependencyID]
		item.Dependencies = append(item.Dependencies, DependencyRef{
			RequestID:                      requestID,
			DependsOnRequestID:             dependencyID,
			RequestUID:                     item.UID,
			DependsOnUID:                   dependency.UID,
			Status:                         dependency.Status,
			StatusReason:                   dependency.StatusReason,
			DeferredUntil:                  cloneTimePtr(dependency.DeferredUntil),
			DependsOnDestinationServerName: dependency.DestinationServerName,
		})
	}
	r.items[requestID] = item
	return nil
}

func (r *memoryRepository) ListDependencies(_ context.Context, requestID int64) ([]DependencyRef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return append([]DependencyRef{}, r.items[requestID].Dependencies...), nil
}

func (r *memoryRepository) ListDependents(_ context.Context, dependencyRequestID int64) ([]DependencyRef, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	items := []DependencyRef{}
	for _, item := range r.items {
		for _, dependency := range item.Dependencies {
			if dependency.DependsOnRequestID == dependencyRequestID {
				items = append(items, dependency)
			}
		}
	}
	return items, nil
}

func (r *memoryRepository) GetDependencyStatuses(_ context.Context, requestID int64) ([]DependencyStatus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	statuses := []DependencyStatus{}
	for _, dependency := range r.items[requestID].Dependencies {
		record := r.items[dependency.DependsOnRequestID]
		statuses = append(statuses, DependencyStatus{RequestID: record.ID, RequestUID: record.UID, Status: record.Status, StatusReason: record.StatusReason})
	}
	return statuses, nil
}

func (r *memoryRepository) DependencyPathExists(_ context.Context, fromRequestID int64, toRequestID int64) (bool, error) {
	return fromRequestID == toRequestID, nil
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
	input.Payload = clonePayloadValue(input.Payload)
	input.LatestDeliveryID = cloneInt64Ptr(input.LatestDeliveryID)
	input.LatestAsyncTaskID = cloneInt64Ptr(input.LatestAsyncTaskID)
	input.DeferredUntil = cloneTimePtr(input.DeferredUntil)
	input.Targets = cloneTargets(input.Targets)
	input.Dependencies = cloneDependencies(input.Dependencies)
	return input
}

func cloneTargets(input []TargetRecord) []TargetRecord {
	items := make([]TargetRecord, 0, len(input))
	for _, item := range input {
		item.DeferredUntil = cloneTimePtr(item.DeferredUntil)
		item.LastReleasedAt = cloneTimePtr(item.LastReleasedAt)
		item.LatestDeliveryID = cloneInt64Ptr(item.LatestDeliveryID)
		item.LatestAsyncTaskID = cloneInt64Ptr(item.LatestAsyncTaskID)
		items = append(items, item)
	}
	return items
}

func cloneDependencies(input []DependencyRef) []DependencyRef {
	items := make([]DependencyRef, 0, len(input))
	for _, item := range input {
		item.DeferredUntil = cloneTimePtr(item.DeferredUntil)
		items = append(items, item)
	}
	return items
}

func cloneInt64Ptr(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneTimePtr(value *time.Time) *time.Time {
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

func clonePayloadValue(input any) any {
	switch value := input.(type) {
	case nil:
		return nil
	case string:
		return value
	default:
		raw, err := json.Marshal(value)
		if err != nil {
			return value
		}
		var cloned any
		if err := json.Unmarshal(raw, &cloned); err != nil {
			return value
		}
		return cloned
	}
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
