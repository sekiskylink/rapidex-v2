package dashboard

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

const (
	attentionLimit            = 5
	workerSummaryLimit        = 10
	workerHeartbeatFreshness  = 2 * time.Minute
	staleRunningDeliveryAfter = 15 * time.Minute
	stuckJobAfter             = 10 * time.Minute
	recentFailureWindow       = time.Hour
	trendWindow               = 24 * time.Hour
	recentIngestFailureWindow = 24 * time.Hour
	graphBucketMinutes        = 60
)

type SQLRepository struct {
	db *sqlx.DB
}

func NewRepository(db *sqlx.DB) *SQLRepository {
	return &SQLRepository{db: db}
}

type kpiRow struct {
	RequestsToday            int `db:"requests_today"`
	PendingRequests          int `db:"pending_requests"`
	PendingDeliveries        int `db:"pending_deliveries"`
	RunningDeliveries        int `db:"running_deliveries"`
	FailedDeliveriesLastHour int `db:"failed_deliveries_last_hour"`
	PollingJobs              int `db:"polling_jobs"`
	IngestBacklog            int `db:"ingest_backlog"`
	HealthyWorkers           int `db:"healthy_workers"`
	UnhealthyWorkers         int `db:"unhealthy_workers"`
}

type deliveryAttentionRow struct {
	TotalCount     int        `db:"total_count"`
	ID             int64      `db:"id"`
	UID            string     `db:"uid"`
	RequestID      int64      `db:"request_id"`
	RequestUID     string     `db:"request_uid"`
	ServerID       int64      `db:"server_id"`
	ServerName     string     `db:"server_name"`
	CorrelationID  string     `db:"correlation_id"`
	Status         string     `db:"status"`
	ErrorMessage   string     `db:"error_message"`
	StartedAt      *time.Time `db:"started_at"`
	FinishedAt     *time.Time `db:"finished_at"`
	NextEligibleAt *time.Time `db:"next_eligible_at"`
	UpdatedAt      time.Time  `db:"updated_at"`
}

type jobAttentionRow struct {
	TotalCount    int        `db:"total_count"`
	ID            int64      `db:"id"`
	UID           string     `db:"uid"`
	DeliveryID    int64      `db:"delivery_id"`
	DeliveryUID   string     `db:"delivery_uid"`
	RequestID     int64      `db:"request_id"`
	RequestUID    string     `db:"request_uid"`
	CorrelationID string     `db:"correlation_id"`
	RemoteJobID   string     `db:"remote_job_id"`
	RemoteStatus  string     `db:"remote_status"`
	CurrentState  string     `db:"current_state"`
	NextPollAt    *time.Time `db:"next_poll_at"`
	UpdatedAt     time.Time  `db:"updated_at"`
}

type ingestAttentionRow struct {
	TotalCount       int        `db:"total_count"`
	ID               int64      `db:"id"`
	UID              string     `db:"uid"`
	OriginalName     string     `db:"original_name"`
	CurrentPath      string     `db:"current_path"`
	Status           string     `db:"status"`
	LastErrorCode    string     `db:"last_error_code"`
	LastErrorMessage string     `db:"last_error_message"`
	RequestID        *int64     `db:"request_id"`
	FailedAt         *time.Time `db:"failed_at"`
	UpdatedAt        time.Time  `db:"updated_at"`
}

type workerRow struct {
	TotalCount      int        `db:"total_count"`
	ID              int64      `db:"id"`
	UID             string     `db:"uid"`
	WorkerType      string     `db:"worker_type"`
	WorkerName      string     `db:"worker_name"`
	Status          string     `db:"status"`
	LastHeartbeatAt *time.Time `db:"last_heartbeat_at"`
	StartedAt       time.Time  `db:"started_at"`
	UpdatedAt       time.Time  `db:"updated_at"`
}

type trendCountRow struct {
	BucketStart time.Time `db:"bucket_start"`
	Count       int       `db:"count"`
}

type trendStatusRow struct {
	BucketStart time.Time `db:"bucket_start"`
	Status      string    `db:"status"`
	Count       int       `db:"count"`
}

type serverCountRow struct {
	ServerID   int64  `db:"server_id"`
	ServerName string `db:"server_name"`
	Count      int    `db:"count"`
}

type eventRow struct {
	EventType     string    `db:"event_type"`
	CreatedAt     time.Time `db:"created_at"`
	EventLevel    string    `db:"event_level"`
	Message       string    `db:"message"`
	CorrelationID string    `db:"correlation_id"`
	RequestID     *int64    `db:"request_id"`
	RequestUID    string    `db:"request_uid"`
	DeliveryID    *int64    `db:"delivery_attempt_id"`
	DeliveryUID   string    `db:"delivery_uid"`
	JobID         *int64    `db:"async_task_id"`
	JobUID        string    `db:"async_task_uid"`
	WorkerID      *int64    `db:"worker_run_id"`
	WorkerUID     string    `db:"worker_run_uid"`
}

func (r *SQLRepository) GetSnapshot(ctx context.Context, now time.Time) (Snapshot, error) {
	if r.db == nil {
		return Snapshot{}, fmt.Errorf("dashboard repository requires database")
	}

	startOfDay := now.Truncate(24 * time.Hour)
	hourAgo := now.Add(-recentFailureWindow)
	workerFreshAfter := now.Add(-workerHeartbeatFreshness)
	staleDeliveryBefore := now.Add(-staleRunningDeliveryAfter)
	stuckJobBefore := now.Add(-stuckJobAfter)
	trendStart := now.Add(-trendWindow)
	recentIngestAfter := now.Add(-recentIngestFailureWindow)

	kpis, err := r.loadKPIs(ctx, startOfDay, hourAgo, workerFreshAfter)
	if err != nil {
		return Snapshot{}, err
	}
	failedDeliveries, err := r.loadFailedDeliveries(ctx, attentionLimit)
	if err != nil {
		return Snapshot{}, err
	}
	staleDeliveries, err := r.loadStaleRunningDeliveries(ctx, staleDeliveryBefore, attentionLimit)
	if err != nil {
		return Snapshot{}, err
	}
	stuckJobs, err := r.loadStuckJobs(ctx, stuckJobBefore, attentionLimit)
	if err != nil {
		return Snapshot{}, err
	}
	ingestFailures, err := r.loadRecentIngestFailures(ctx, recentIngestAfter, attentionLimit)
	if err != nil {
		return Snapshot{}, err
	}
	unhealthyWorkers, err := r.loadUnhealthyWorkers(ctx, workerFreshAfter, attentionLimit)
	if err != nil {
		return Snapshot{}, err
	}
	workerItems, err := r.loadWorkers(ctx, workerSummaryLimit)
	if err != nil {
		return Snapshot{}, err
	}
	trends, err := r.loadTrends(ctx, trendStart)
	if err != nil {
		return Snapshot{}, err
	}
	recentEvents, err := r.loadRecentEvents(ctx, workerSummaryLimit)
	if err != nil {
		return Snapshot{}, err
	}
	graphEvents, err := r.loadGraphEvents(ctx, trendStart)
	if err != nil {
		return Snapshot{}, err
	}

	snapshot := Snapshot{
		GeneratedAt:     now,
		KPIs:            kpis,
		Trends:          trends,
		ProcessingGraph: loadProcessingGraph(now, trendStart, graphEvents),
		Attention: Attention{
			FailedDeliveries:       failedDeliveries,
			StaleRunningDeliveries: staleDeliveries,
			StuckJobs:              stuckJobs,
			RecentIngestFailures:   ingestFailures,
			UnhealthyWorkers:       unhealthyWorkers,
		},
		Workers: WorkersSummary{
			HeartbeatFreshnessSeconds: int(workerHeartbeatFreshness / time.Second),
			Items:                     workerItems,
		},
		RecentEvents: recentEvents,
	}
	snapshot.Health = deriveHealth(snapshot)
	return snapshot, nil
}

func (r *SQLRepository) loadKPIs(ctx context.Context, startOfDay, hourAgo, workerFreshAfter time.Time) (KPIs, error) {
	var row kpiRow
	if err := r.db.GetContext(ctx, &row, `
		SELECT
			(SELECT COUNT(*) FROM exchange_requests WHERE created_at >= $1) AS requests_today,
			(SELECT COUNT(*) FROM exchange_requests WHERE status IN ('pending', 'blocked')) AS pending_requests,
			(SELECT COUNT(*) FROM delivery_attempts WHERE status IN ('pending', 'retrying')) AS pending_deliveries,
			(SELECT COUNT(*) FROM delivery_attempts WHERE status = 'running') AS running_deliveries,
			(SELECT COUNT(*) FROM delivery_attempts WHERE status = 'failed' AND COALESCE(finished_at, updated_at, created_at) >= $2) AS failed_deliveries_last_hour,
			(SELECT COUNT(*) FROM async_tasks WHERE COALESCE(terminal_state, '') = '' AND LOWER(COALESCE(remote_status, '')) = 'polling') AS polling_jobs,
			(SELECT COUNT(*) FROM ingest_files WHERE status IN ('discovered', 'retry', 'processing')) AS ingest_backlog,
			(SELECT COUNT(*) FROM worker_runs WHERE status = 'running' AND COALESCE(last_heartbeat_at, started_at) >= $3) AS healthy_workers,
			(SELECT COUNT(*) FROM worker_runs WHERE NOT (status = 'running' AND COALESCE(last_heartbeat_at, started_at) >= $3)) AS unhealthy_workers
	`, startOfDay, hourAgo, workerFreshAfter); err != nil {
		return KPIs{}, fmt.Errorf("load dashboard kpis: %w", err)
	}
	return KPIs(row), nil
}

func (r *SQLRepository) loadFailedDeliveries(ctx context.Context, limit int) (DeliveryAttentionList, error) {
	rows := []deliveryAttentionRow{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT COUNT(*) OVER() AS total_count, d.id, d.uid::text AS uid, d.request_id,
		       COALESCE(req.uid::text, '') AS request_uid, d.server_id, COALESCE(s.name, '') AS server_name,
		       COALESCE(req.correlation_id, '') AS correlation_id, d.status, COALESCE(d.error_message, '') AS error_message,
		       d.started_at, d.finished_at, d.next_eligible_at, d.updated_at
		FROM delivery_attempts d
		JOIN exchange_requests req ON req.id = d.request_id
		JOIN integration_servers s ON s.id = d.server_id
		WHERE d.status = 'failed'
		ORDER BY COALESCE(d.finished_at, d.updated_at, d.created_at) DESC, d.id DESC
		LIMIT $1
	`, limit); err != nil {
		return DeliveryAttentionList{}, fmt.Errorf("load failed deliveries attention: %w", err)
	}
	return mapDeliveryAttention(rows), nil
}

func (r *SQLRepository) loadStaleRunningDeliveries(ctx context.Context, staleBefore time.Time, limit int) (DeliveryAttentionList, error) {
	rows := []deliveryAttentionRow{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT COUNT(*) OVER() AS total_count, d.id, d.uid::text AS uid, d.request_id,
		       COALESCE(req.uid::text, '') AS request_uid, d.server_id, COALESCE(s.name, '') AS server_name,
		       COALESCE(req.correlation_id, '') AS correlation_id, d.status, COALESCE(d.error_message, '') AS error_message,
		       d.started_at, d.finished_at, d.next_eligible_at, d.updated_at
		FROM delivery_attempts d
		JOIN exchange_requests req ON req.id = d.request_id
		JOIN integration_servers s ON s.id = d.server_id
		WHERE d.status = 'running'
		  AND COALESCE(d.started_at, d.created_at) < $1
		ORDER BY COALESCE(d.started_at, d.created_at) ASC, d.id ASC
		LIMIT $2
	`, staleBefore, limit); err != nil {
		return DeliveryAttentionList{}, fmt.Errorf("load stale running deliveries attention: %w", err)
	}
	return mapDeliveryAttention(rows), nil
}

func (r *SQLRepository) loadStuckJobs(ctx context.Context, stuckBefore time.Time, limit int) (JobAttentionList, error) {
	rows := []jobAttentionRow{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT COUNT(*) OVER() AS total_count, a.id, a.uid::text AS uid, d.id AS delivery_id,
		       COALESCE(d.uid::text, '') AS delivery_uid, req.id AS request_id, COALESCE(req.uid::text, '') AS request_uid,
		       COALESCE(req.correlation_id, '') AS correlation_id, COALESCE(a.remote_job_id, '') AS remote_job_id,
		       COALESCE(a.remote_status, '') AS remote_status,
		       CASE
		         WHEN COALESCE(a.terminal_state, '') <> '' THEN a.terminal_state
		         WHEN COALESCE(a.remote_status, '') <> '' THEN a.remote_status
		         ELSE 'pending'
		       END AS current_state,
		       a.next_poll_at, a.updated_at
		FROM async_tasks a
		JOIN delivery_attempts d ON d.id = a.delivery_attempt_id
		JOIN exchange_requests req ON req.id = d.request_id
		WHERE COALESCE(a.terminal_state, '') = ''
		  AND LOWER(COALESCE(a.remote_status, '')) = 'polling'
		  AND COALESCE(a.next_poll_at, a.updated_at, a.created_at) < $1
		ORDER BY COALESCE(a.next_poll_at, a.updated_at, a.created_at) ASC, a.id ASC
		LIMIT $2
	`, stuckBefore, limit); err != nil {
		return JobAttentionList{}, fmt.Errorf("load stuck jobs attention: %w", err)
	}
	return mapJobAttention(rows), nil
}

func (r *SQLRepository) loadRecentIngestFailures(ctx context.Context, recentAfter time.Time, limit int) (IngestAttentionList, error) {
	rows := []ingestAttentionRow{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT COUNT(*) OVER() AS total_count, i.id, i.uid::text AS uid, i.original_name, i.current_path, i.status,
		       COALESCE(i.last_error_code, '') AS last_error_code, COALESCE(i.last_error_message, '') AS last_error_message,
		       i.request_id, i.failed_at, i.updated_at
		FROM ingest_files i
		WHERE i.status IN ('failed', 'retry')
		  AND COALESCE(i.failed_at, i.updated_at, i.created_at) >= $1
		ORDER BY COALESCE(i.failed_at, i.updated_at, i.created_at) DESC, i.id DESC
		LIMIT $2
	`, recentAfter, limit); err != nil {
		return IngestAttentionList{}, fmt.Errorf("load ingest failures attention: %w", err)
	}
	return mapIngestAttention(rows), nil
}

func (r *SQLRepository) loadUnhealthyWorkers(ctx context.Context, workerFreshAfter time.Time, limit int) (WorkerAttentionList, error) {
	rows := []workerRow{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT COUNT(*) OVER() AS total_count, w.id, w.uid::text AS uid, w.worker_type, w.worker_name, w.status,
		       w.last_heartbeat_at, w.started_at, w.updated_at
		FROM worker_runs w
		WHERE NOT (w.status = 'running' AND COALESCE(w.last_heartbeat_at, w.started_at) >= $1)
		ORDER BY w.updated_at DESC, w.id DESC
		LIMIT $2
	`, workerFreshAfter, limit); err != nil {
		return WorkerAttentionList{}, fmt.Errorf("load unhealthy workers attention: %w", err)
	}
	return mapWorkerAttention(rows), nil
}

func (r *SQLRepository) loadWorkers(ctx context.Context, limit int) ([]WorkerAttentionItem, error) {
	rows := []workerRow{}
	if err := r.db.SelectContext(ctx, &rows, `
		SELECT 0 AS total_count, w.id, w.uid::text AS uid, w.worker_type, w.worker_name, w.status,
		       w.last_heartbeat_at, w.started_at, w.updated_at
		FROM worker_runs w
		ORDER BY w.updated_at DESC, w.id DESC
		LIMIT $1
	`, limit); err != nil {
		return nil, fmt.Errorf("load dashboard workers: %w", err)
	}
	items := make([]WorkerAttentionItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, WorkerAttentionItem{
			ID:              row.ID,
			UID:             row.UID,
			WorkerType:      row.WorkerType,
			WorkerName:      row.WorkerName,
			Status:          row.Status,
			LastHeartbeatAt: row.LastHeartbeatAt,
			StartedAt:       row.StartedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}
	return items, nil
}

func (r *SQLRepository) loadTrends(ctx context.Context, trendStart time.Time) (Trends, error) {
	requestRows := []trendCountRow{}
	if err := r.db.SelectContext(ctx, &requestRows, `
		SELECT date_trunc('hour', created_at) AS bucket_start, COUNT(*) AS count
		FROM exchange_requests
		WHERE created_at >= $1
		GROUP BY 1
		ORDER BY 1 ASC
	`, trendStart); err != nil {
		return Trends{}, fmt.Errorf("load request trend: %w", err)
	}

	deliveryRows := []trendStatusRow{}
	if err := r.db.SelectContext(ctx, &deliveryRows, `
		SELECT date_trunc('hour', COALESCE(finished_at, started_at, created_at)) AS bucket_start, status, COUNT(*) AS count
		FROM delivery_attempts
		WHERE COALESCE(finished_at, started_at, created_at) >= $1
		GROUP BY 1, 2
		ORDER BY 1 ASC, 2 ASC
	`, trendStart); err != nil {
		return Trends{}, fmt.Errorf("load delivery trend: %w", err)
	}

	jobRows := []trendStatusRow{}
	if err := r.db.SelectContext(ctx, &jobRows, `
		SELECT date_trunc('hour', COALESCE(completed_at, next_poll_at, updated_at, created_at)) AS bucket_start,
		       CASE
		         WHEN COALESCE(terminal_state, '') <> '' THEN terminal_state
		         WHEN COALESCE(remote_status, '') <> '' THEN remote_status
		         ELSE 'pending'
		       END AS status,
		       COUNT(*) AS count
		FROM async_tasks
		WHERE COALESCE(completed_at, next_poll_at, updated_at, created_at) >= $1
		GROUP BY 1, 2
		ORDER BY 1 ASC, 2 ASC
	`, trendStart); err != nil {
		return Trends{}, fmt.Errorf("load job trend: %w", err)
	}

	serverRows := []serverCountRow{}
	if err := r.db.SelectContext(ctx, &serverRows, `
		SELECT s.id AS server_id, COALESCE(s.name, '') AS server_name, COUNT(*) AS count
		FROM delivery_attempts d
		JOIN integration_servers s ON s.id = d.server_id
		WHERE d.status = 'failed'
		  AND COALESCE(d.finished_at, d.updated_at, d.created_at) >= $1
		GROUP BY s.id, s.name
		ORDER BY count DESC, s.name ASC
		LIMIT 5
	`, trendStart); err != nil {
		return Trends{}, fmt.Errorf("load failures by server trend: %w", err)
	}

	return Trends{
		RequestsByHour:     mapTrendCounts(requestRows),
		DeliveriesByStatus: mapTrendStatuses(deliveryRows),
		JobsByState:        mapTrendStatuses(jobRows),
		FailuresByServer:   mapServerCounts(serverRows),
	}, nil
}

func loadProcessingGraph(now, trendStart time.Time, events []EventSummary) ProcessingGraph {
	seriesMap := make(map[time.Time]*ProcessingGraphStage)
	for bucket := trendStart.Truncate(time.Hour); !bucket.After(now); bucket = bucket.Add(time.Hour) {
		stage := ProcessingGraphStage{}
		seriesMap[bucket] = &stage
	}

	for _, event := range events {
		stageName, ok := stageForEvent(event)
		if !ok {
			continue
		}
		bucketStart := event.Timestamp.UTC().Truncate(time.Hour)
		stage, ok := seriesMap[bucketStart]
		if !ok {
			continue
		}
		switch stageName {
		case "pending":
			stage.Pending++
		case "processing":
			stage.Processing++
		case "completed":
			stage.Completed++
		case "failed":
			stage.Failed++
		}
	}

	series := make([]ProcessingGraphPoint, 0, len(seriesMap))
	for bucket := trendStart.Truncate(time.Hour); !bucket.After(now); bucket = bucket.Add(time.Hour) {
		stage := seriesMap[bucket]
		series = append(series, ProcessingGraphPoint{
			BucketStart: bucket,
			Stages:      *stage,
		})
	}

	return ProcessingGraph{
		BucketSizeMinutes: graphBucketMinutes,
		WindowHours:       int(trendWindow / time.Hour),
		Series:            series,
	}
}

func (r *SQLRepository) loadRecentEvents(ctx context.Context, limit int) ([]EventSummary, error) {
	rows := []eventRow{}
	if err := r.db.SelectContext(ctx, &rows, recentEventsQuery(limit), limit); err != nil {
		return nil, fmt.Errorf("load dashboard recent events: %w", err)
	}
	return mapEventSummaries(rows), nil
}

func (r *SQLRepository) loadGraphEvents(ctx context.Context, trendStart time.Time) ([]EventSummary, error) {
	rows := []eventRow{}
	if err := r.db.SelectContext(ctx, &rows, recentEventsQuery(0)+`
		WHERE e.created_at >= $1
		  AND e.event_type IN ('request.created', 'request.status_changed', 'request.completed', 'request.failed')
		ORDER BY e.created_at ASC, e.id ASC
	`, trendStart); err != nil {
		return nil, fmt.Errorf("load dashboard graph events: %w", err)
	}
	return mapEventSummaries(rows), nil
}

func recentEventsQuery(limit int) string {
	query := `
		SELECT e.event_type, e.created_at, e.event_level, COALESCE(e.message, '') AS message, COALESCE(e.correlation_id, '') AS correlation_id,
		       e.request_id, COALESCE(req.uid::text, '') AS request_uid, e.delivery_attempt_id, COALESCE(d.uid::text, '') AS delivery_uid,
		       e.async_task_id, COALESCE(a.uid::text, '') AS async_task_uid, e.worker_run_id, COALESCE(w.uid::text, '') AS worker_run_uid
		FROM request_events e
		LEFT JOIN exchange_requests req ON req.id = e.request_id
		LEFT JOIN delivery_attempts d ON d.id = e.delivery_attempt_id
		LEFT JOIN async_tasks a ON a.id = e.async_task_id
		LEFT JOIN worker_runs w ON w.id = e.worker_run_id
	`
	if limit > 0 {
		query += `
		ORDER BY e.created_at DESC, e.id DESC
		LIMIT $1
	`
	}
	return query
}

func mapEventSummaries(rows []eventRow) []EventSummary {
	items := make([]EventSummary, 0, len(rows))
	for _, row := range rows {
		item := EventSummary{
			Type:          row.EventType,
			Timestamp:     row.CreatedAt,
			Severity:      row.EventLevel,
			Summary:       strings.TrimSpace(row.Message),
			CorrelationID: row.CorrelationID,
			RequestID:     row.RequestID,
			DeliveryID:    row.DeliveryID,
			JobID:         row.JobID,
			WorkerID:      row.WorkerID,
		}
		if item.Summary == "" {
			item.Summary = row.EventType
		}
		switch {
		case row.DeliveryID != nil:
			item.EntityType = "delivery"
			item.EntityID = *row.DeliveryID
			item.EntityUID = row.DeliveryUID
		case row.JobID != nil:
			item.EntityType = "job"
			item.EntityID = *row.JobID
			item.EntityUID = row.JobUID
		case row.WorkerID != nil:
			item.EntityType = "worker"
			item.EntityID = *row.WorkerID
			item.EntityUID = row.WorkerUID
		case row.RequestID != nil:
			item.EntityType = "request"
			item.EntityID = *row.RequestID
			item.EntityUID = row.RequestUID
		}
		items = append(items, item)
	}
	return items
}

func stageForEvent(event EventSummary) (string, bool) {
	switch event.Type {
	case "request.created", "request.submitted":
		return "pending", true
	case "request.completed":
		return "completed", true
	case "request.failed":
		return "failed", true
	case "request.status_changed":
		summary := strings.ToLower(event.Summary)
		switch {
		case strings.Contains(summary, " to processing"):
			return "processing", true
		case strings.Contains(summary, " to blocked"), strings.Contains(summary, " to pending"):
			return "pending", true
		case strings.Contains(summary, " to completed"):
			return "completed", true
		case strings.Contains(summary, " to failed"):
			return "failed", true
		}
	}
	return "", false
}

func deriveHealth(snapshot Snapshot) Health {
	signals := make([]string, 0, 4)
	if snapshot.Attention.StaleRunningDeliveries.Total > 0 {
		signals = append(signals, "stale running deliveries detected")
	}
	if snapshot.Attention.StuckJobs.Total > 0 {
		signals = append(signals, "stuck polling jobs detected")
	}
	if snapshot.Attention.RecentIngestFailures.Total > 0 {
		signals = append(signals, "recent ingest failures detected")
	}
	if snapshot.KPIs.UnhealthyWorkers > 0 {
		signals = append(signals, "unhealthy workers detected")
	}
	status := "ok"
	if len(signals) > 0 {
		status = "degraded"
	}
	return Health{Status: status, Signals: signals}
}

func mapDeliveryAttention(rows []deliveryAttentionRow) DeliveryAttentionList {
	items := make([]DeliveryAttentionItem, 0, len(rows))
	total := 0
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		items = append(items, DeliveryAttentionItem{
			ID:             row.ID,
			UID:            row.UID,
			RequestID:      row.RequestID,
			RequestUID:     row.RequestUID,
			ServerID:       row.ServerID,
			ServerName:     row.ServerName,
			CorrelationID:  row.CorrelationID,
			Status:         row.Status,
			ErrorMessage:   row.ErrorMessage,
			StartedAt:      row.StartedAt,
			FinishedAt:     row.FinishedAt,
			NextEligibleAt: row.NextEligibleAt,
			UpdatedAt:      row.UpdatedAt,
		})
	}
	return DeliveryAttentionList{Total: total, Items: items}
}

func mapJobAttention(rows []jobAttentionRow) JobAttentionList {
	items := make([]JobAttentionItem, 0, len(rows))
	total := 0
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		items = append(items, JobAttentionItem{
			ID:            row.ID,
			UID:           row.UID,
			DeliveryID:    row.DeliveryID,
			DeliveryUID:   row.DeliveryUID,
			RequestID:     row.RequestID,
			RequestUID:    row.RequestUID,
			CorrelationID: row.CorrelationID,
			RemoteJobID:   row.RemoteJobID,
			RemoteStatus:  row.RemoteStatus,
			CurrentState:  row.CurrentState,
			NextPollAt:    row.NextPollAt,
			UpdatedAt:     row.UpdatedAt,
		})
	}
	return JobAttentionList{Total: total, Items: items}
}

func mapIngestAttention(rows []ingestAttentionRow) IngestAttentionList {
	items := make([]IngestAttentionItem, 0, len(rows))
	total := 0
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		items = append(items, IngestAttentionItem{
			ID:               row.ID,
			UID:              row.UID,
			OriginalName:     row.OriginalName,
			CurrentPath:      row.CurrentPath,
			Status:           row.Status,
			LastErrorCode:    row.LastErrorCode,
			LastErrorMessage: row.LastErrorMessage,
			RequestID:        row.RequestID,
			FailedAt:         row.FailedAt,
			UpdatedAt:        row.UpdatedAt,
		})
	}
	return IngestAttentionList{Total: total, Items: items}
}

func mapWorkerAttention(rows []workerRow) WorkerAttentionList {
	items := make([]WorkerAttentionItem, 0, len(rows))
	total := 0
	for _, row := range rows {
		if total == 0 {
			total = row.TotalCount
		}
		items = append(items, WorkerAttentionItem{
			ID:              row.ID,
			UID:             row.UID,
			WorkerType:      row.WorkerType,
			WorkerName:      row.WorkerName,
			Status:          row.Status,
			LastHeartbeatAt: row.LastHeartbeatAt,
			StartedAt:       row.StartedAt,
			UpdatedAt:       row.UpdatedAt,
		})
	}
	return WorkerAttentionList{Total: total, Items: items}
}

func mapTrendCounts(rows []trendCountRow) []TimeCountPoint {
	items := make([]TimeCountPoint, 0, len(rows))
	for _, row := range rows {
		items = append(items, TimeCountPoint{BucketStart: row.BucketStart, Count: row.Count})
	}
	return items
}

func mapTrendStatuses(rows []trendStatusRow) []StatusCountPoint {
	items := make([]StatusCountPoint, 0, len(rows))
	for _, row := range rows {
		items = append(items, StatusCountPoint{BucketStart: row.BucketStart, Status: row.Status, Count: row.Count})
	}
	return items
}

func mapServerCounts(rows []serverCountRow) []ServerCountPoint {
	items := make([]ServerCountPoint, 0, len(rows))
	for _, row := range rows {
		items = append(items, ServerCountPoint{ServerID: row.ServerID, ServerName: row.ServerName, Count: row.Count})
	}
	return items
}
