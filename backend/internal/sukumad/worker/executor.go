package worker

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"basepro/backend/internal/logging"
	"basepro/backend/internal/sukumad/delivery"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/traceevent"
)

type DeliveryExecutor struct {
	deliveryRepo interface {
		ClaimNextPendingDelivery(context.Context, time.Time) (delivery.Record, error)
		ClaimNextRetryDelivery(context.Context, time.Time) (delivery.Record, error)
	}
	requestService interface {
		GetRequest(context.Context, int64) (requests.Record, error)
	}
	serverService interface {
		GetServer(context.Context, int64) (server.Record, error)
	}
	deliveryService interface {
		SubmitDHIS2Delivery(context.Context, delivery.DispatchInput) (delivery.Record, error)
		RecoverStaleRunningDeliveries(context.Context, time.Time) ([]delivery.Record, error)
	}
	eventWriter traceevent.Writer
	now         func() time.Time
}

func NewDeliveryExecutor(
	deliveryRepo interface {
		ClaimNextPendingDelivery(context.Context, time.Time) (delivery.Record, error)
		ClaimNextRetryDelivery(context.Context, time.Time) (delivery.Record, error)
	},
	requestService interface {
		GetRequest(context.Context, int64) (requests.Record, error)
	},
	serverService interface {
		GetServer(context.Context, int64) (server.Record, error)
	},
	deliveryService interface {
		SubmitDHIS2Delivery(context.Context, delivery.DispatchInput) (delivery.Record, error)
		RecoverStaleRunningDeliveries(context.Context, time.Time) ([]delivery.Record, error)
	},
) *DeliveryExecutor {
	return &DeliveryExecutor{
		deliveryRepo:    deliveryRepo,
		requestService:  requestService,
		serverService:   serverService,
		deliveryService: deliveryService,
		now: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (e *DeliveryExecutor) WithEventWriter(eventWriter traceevent.Writer) *DeliveryExecutor {
	e.eventWriter = eventWriter
	return e
}

func (e *DeliveryExecutor) RunSendBatch(ctx context.Context, exec Execution, batchSize int) error {
	return e.runBatch(ctx, exec, batchSize, false)
}

func (e *DeliveryExecutor) RunRetryBatch(ctx context.Context, exec Execution, batchSize int) error {
	return e.runBatch(ctx, exec, batchSize, true)
}

func (e *DeliveryExecutor) runBatch(ctx context.Context, exec Execution, batchSize int, retry bool) error {
	if e == nil || e.deliveryRepo == nil || e.requestService == nil || e.serverService == nil || e.deliveryService == nil {
		return nil
	}
	if batchSize <= 0 {
		batchSize = 10
	}
	for i := 0; i < batchSize; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		record, err := e.claim(ctx, retry)
		if err != nil {
			if errors.Is(err, delivery.ErrNoEligibleDelivery) {
				return nil
			}
			logging.ForContext(ctx).Error("delivery_worker_error",
				slog.Int64("worker_run_id", exec.RunID),
				slog.String("worker_type", workerType(retry)),
				slog.String("stage", "claim"),
				slog.String("error", safeError(err)),
			)
			return err
		}
		requestRecord, err := e.requestService.GetRequest(ctx, record.RequestID)
		if err != nil {
			e.logDeliveryWorker(ctx, "delivery_worker_error", "error", exec, record, retry, nil, 0,
				slog.String("stage", "load_request"),
				slog.String("error", safeError(err)),
			)
			return err
		}
		serverRecord, err := e.serverService.GetServer(ctx, record.ServerID)
		if err != nil {
			e.logDeliveryWorker(ctx, "delivery_worker_error", "error", exec, record, retry, nil, 0,
				slog.String("stage", "load_server"),
				slog.String("error", safeError(err)),
			)
			return err
		}
		e.logDeliveryWorker(ctx, "delivery_worker_picked", "info", exec, record, retry, &serverRecord, 0)
		e.appendWorkerEvent(ctx, exec.RunID, record, retry, "delivery.worker.picked", "info", "Worker picked delivery", map[string]any{
			"deliveryUid": record.UID,
			"requestUid":  requestRecord.UID,
			"workerType":  workerType(retry),
		})
		exec.Increment("deliveries_picked")
		e.appendWorkerEvent(ctx, exec.RunID, record, retry, "delivery.worker.claimed", "info", "Worker claimed delivery", map[string]any{
			"deliveryUid": record.UID,
			"requestUid":  requestRecord.UID,
			"workerType":  workerType(retry),
			"status":      record.Status,
		})
		e.appendWorkerEvent(ctx, exec.RunID, record, retry, "delivery.submission.started", "info", "Delivery submission started", map[string]any{
			"deliveryUid": record.UID,
			"requestUid":  requestRecord.UID,
			"serverCode":  serverRecord.Code,
			"workerType":  workerType(retry),
		})
		startedAt := time.Now()
		submitted, err := e.deliveryService.SubmitDHIS2Delivery(ctx, delivery.DispatchInput{
			DeliveryID:              record.ID,
			RequestID:               requestRecord.ID,
			RequestUID:              requestRecord.UID,
			CorrelationID:           requestRecord.CorrelationID,
			PayloadBody:             requestRecord.PayloadBody,
			PayloadFormat:           requestRecord.PayloadFormat,
			SubmissionBinding:       requestRecord.SubmissionBinding,
			ResponseBodyPersistence: requestRecord.ResponseBodyPersistence,
			URLSuffix:               requestRecord.URLSuffix,
			Server: delivery.ServerSnapshot{
				ID:                      serverRecord.ID,
				Code:                    serverRecord.Code,
				Name:                    serverRecord.Name,
				SystemType:              serverRecord.SystemType,
				BaseURL:                 serverRecord.BaseURL,
				HTTPMethod:              serverRecord.HTTPMethod,
				UseAsync:                serverRecord.UseAsync,
				ResponseBodyPersistence: serverRecord.ResponseBodyPersistence,
				Headers:                 cloneServerMap(serverRecord.Headers),
				URLParams:               cloneServerMap(serverRecord.URLParams),
			},
		})
		if err != nil {
			e.logDeliveryWorker(ctx, "delivery_worker_error", "error", exec, record, retry, &serverRecord, time.Since(startedAt),
				slog.String("stage", "submit"),
				slog.String("error", safeError(err)),
			)
			return err
		}
		eventType := "delivery.submission.completed"
		eventLevel := "info"
		message := "Delivery submission completed"
		data := map[string]any{
			"deliveryUid": submitted.UID,
			"requestUid":  requestRecord.UID,
			"serverCode":  serverRecord.Code,
			"workerType":  workerType(retry),
			"status":      submitted.Status,
		}
		if submitted.Status == delivery.StatusPending && submitted.SubmissionHoldReason != "" {
			eventType = "delivery.submission.deferred"
			message = "Delivery submission deferred"
			data["deferReason"] = submitted.SubmissionHoldReason
			data["nextEligibleAt"] = submitted.NextEligibleAt
		}
		if retry {
			eventType = "delivery.retry.executed"
			message = "Delivery retry executed"
			if submitted.Status == delivery.StatusPending && submitted.SubmissionHoldReason != "" {
				eventType = "delivery.retry.deferred"
				message = "Delivery retry deferred"
			}
		}
		if submitted.Status == delivery.StatusFailed {
			eventLevel = "warning"
			exec.Increment("deliveries_failed")
		} else if submitted.Status == delivery.StatusPending && submitted.SubmissionHoldReason != "" {
			exec.Increment("deliveries_deferred")
		} else {
			exec.Increment("deliveries_completed")
		}
		if retry {
			exec.Increment("retries_executed")
		}
		logEvent := "delivery_worker_completed"
		logLevel := "info"
		if submitted.Status == delivery.StatusFailed {
			logEvent = "delivery_worker_failed"
			logLevel = "warn"
		} else if submitted.Status == delivery.StatusPending && submitted.SubmissionHoldReason != "" {
			logEvent = "delivery_worker_deferred"
		}
		e.logDeliveryWorker(ctx, logEvent, logLevel, exec, submitted, retry, &serverRecord, time.Since(startedAt))
		e.appendWorkerEvent(ctx, exec.RunID, submitted, retry, eventType, eventLevel, message, data)
	}
	return nil
}

func (e *DeliveryExecutor) RecoverStaleRunning(ctx context.Context, exec Execution, staleAfter time.Duration) error {
	if e == nil || e.deliveryService == nil || staleAfter <= 0 {
		return nil
	}
	cutoff := e.now().Add(-staleAfter)
	recovered, err := e.deliveryService.RecoverStaleRunningDeliveries(ctx, cutoff)
	if err != nil {
		return err
	}
	for _, record := range recovered {
		exec.Increment("stale_running_recovered")
		e.appendWorkerEvent(ctx, exec.RunID, record, record.AttemptNumber > 1, "delivery.recovered.stale_running", "warning", "Recovered stale running delivery", map[string]any{
			"deliveryUid": record.UID,
			"requestUid":  record.RequestUID,
			"recoverTo":   record.Status,
		})
	}
	return nil
}

func (e *DeliveryExecutor) claim(ctx context.Context, retry bool) (delivery.Record, error) {
	now := time.Now().UTC()
	if e.now != nil {
		now = e.now()
	}
	if retry {
		return e.deliveryRepo.ClaimNextRetryDelivery(ctx, now)
	}
	return e.deliveryRepo.ClaimNextPendingDelivery(ctx, now)
}

func (e *DeliveryExecutor) appendWorkerEvent(ctx context.Context, runID int64, record delivery.Record, retry bool, eventType string, eventLevel string, message string, data map[string]any) {
	if e.eventWriter == nil {
		return
	}
	workerRunID := runID
	requestID := record.RequestID
	deliveryID := record.ID
	_ = e.eventWriter.AppendEvent(ctx, traceevent.WriteInput{
		RequestID:         &requestID,
		DeliveryAttemptID: &deliveryID,
		WorkerRunID:       &workerRunID,
		EventType:         eventType,
		EventLevel:        eventLevel,
		Message:           message,
		Actor: traceevent.Actor{
			Type: traceevent.ActorWorker,
			Name: fmt.Sprintf("%s-worker", workerType(retry)),
		},
		SourceComponent: "worker.delivery_executor",
		CorrelationID:   record.CorrelationID,
		EventData:       data,
	})
}

func (e *DeliveryExecutor) logDeliveryWorker(ctx context.Context, event string, level string, exec Execution, record delivery.Record, retry bool, serverRecord *server.Record, duration time.Duration, extra ...slog.Attr) {
	serverID := record.ServerID
	serverCode := record.ServerCode
	if serverRecord != nil {
		serverID = serverRecord.ID
		serverCode = serverRecord.Code
	}
	attrs := []slog.Attr{
		slog.Int64("worker_run_id", exec.RunID),
		slog.String("worker_type", workerType(retry)),
		slog.Int64("request_id", record.RequestID),
		slog.String("request_uid", record.RequestUID),
		slog.Int64("delivery_attempt_id", record.ID),
		slog.String("delivery_uid", record.UID),
		slog.String("correlation_id", record.CorrelationID),
		slog.Int64("server_id", serverID),
		slog.String("server_code", serverCode),
		slog.Int("attempt_number", record.AttemptNumber),
		slog.String("status", record.Status),
		intPtrAttr("http_status", record.HTTPStatus),
	}
	if duration > 0 {
		attrs = append(attrs, slog.Int64("duration_ms", duration.Milliseconds()))
	}
	attrs = append(attrs, extra...)

	logger := logging.ForContext(ctx)
	switch level {
	case "error":
		logger.LogAttrs(ctx, slog.LevelError, event, attrs...)
	case "warn":
		logger.LogAttrs(ctx, slog.LevelWarn, event, attrs...)
	default:
		logger.LogAttrs(ctx, slog.LevelInfo, event, attrs...)
	}
}

func intPtrAttr(name string, value *int) slog.Attr {
	if value == nil {
		return slog.Any(name, nil)
	}
	return slog.Int(name, *value)
}

func safeError(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func workerType(retry bool) string {
	if retry {
		return TypeRetry
	}
	return TypeSend
}

func cloneServerMap(input map[string]string) map[string]string {
	if len(input) == 0 {
		return map[string]string{}
	}
	cloned := make(map[string]string, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
