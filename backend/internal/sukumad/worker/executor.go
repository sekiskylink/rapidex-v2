package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

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
			return err
		}
		requestRecord, err := e.requestService.GetRequest(ctx, record.RequestID)
		if err != nil {
			return err
		}
		serverRecord, err := e.serverService.GetServer(ctx, record.ServerID)
		if err != nil {
			return err
		}
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
		submitted, err := e.deliveryService.SubmitDHIS2Delivery(ctx, delivery.DispatchInput{
			DeliveryID:    record.ID,
			RequestID:     requestRecord.ID,
			RequestUID:    requestRecord.UID,
			CorrelationID: requestRecord.CorrelationID,
			PayloadBody:   requestRecord.PayloadBody,
			URLSuffix:     requestRecord.URLSuffix,
			Server: delivery.ServerSnapshot{
				ID:         serverRecord.ID,
				Code:       serverRecord.Code,
				Name:       serverRecord.Name,
				SystemType: serverRecord.SystemType,
				BaseURL:    serverRecord.BaseURL,
				HTTPMethod: serverRecord.HTTPMethod,
				UseAsync:   serverRecord.UseAsync,
				Headers:    cloneServerMap(serverRecord.Headers),
				URLParams:  cloneServerMap(serverRecord.URLParams),
			},
		})
		if err != nil {
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
