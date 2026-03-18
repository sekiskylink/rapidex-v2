package dashboard

import (
	"context"
	"testing"
	"time"
)

func TestServicePublishSourceEventMapsDeliveryFailure(t *testing.T) {
	service := NewService(&stubRepository{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, unsubscribe := service.SubscribeOperationsEvents(ctx)
	defer unsubscribe()

	requestID := int64(12)
	deliveryID := int64(42)
	serverID := int64(3)
	service.PublishSourceEvent(context.Background(), SourceEvent{
		Type:          "delivery.failed",
		Timestamp:     time.Date(2026, 3, 18, 12, 30, 0, 0, time.UTC),
		Severity:      "error",
		Message:       "Delivery to DHIS2 Uganda failed",
		CorrelationID: "corr-456",
		RequestID:     &requestID,
		DeliveryID:    &deliveryID,
		DeliveryUID:   "del_123",
		Payload: map[string]any{
			"serverId":   serverID,
			"serverName": "DHIS2 Uganda",
			"status":     "failed",
		},
	})

	select {
	case event := <-events:
		if event.EntityType != "delivery" || event.EntityID != deliveryID || event.EntityUID != "del_123" {
			t.Fatalf("unexpected entity mapping: %+v", event)
		}
		if event.Patch == nil || event.Patch.KPI != "failedDeliveriesLastHour" || event.Patch.Op != "increment" || event.Patch.Value != 1 {
			t.Fatalf("unexpected patch: %+v", event.Patch)
		}
		if event.ServerID == nil || *event.ServerID != serverID {
			t.Fatalf("expected server id %d, got %+v", serverID, event.ServerID)
		}
		if len(event.Invalidations) == 0 {
			t.Fatalf("expected invalidations, got %+v", event)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream event")
	}
}

func TestServicePublishSourceEventSanitizesPayload(t *testing.T) {
	service := NewService(&stubRepository{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	events, unsubscribe := service.SubscribeOperationsEvents(ctx)
	defer unsubscribe()

	service.PublishSourceEvent(context.Background(), SourceEvent{
		Type:      "worker.error",
		Timestamp: time.Date(2026, 3, 18, 13, 0, 0, 0, time.UTC),
		Severity:  "error",
		Message:   "Worker failed",
		Payload: map[string]any{
			"accessToken": "secret",
			"nested": map[string]any{
				"password": "hidden",
			},
			"status": "failed",
		},
	})

	select {
	case event := <-events:
		if event.Payload["accessToken"] != "[masked]" {
			t.Fatalf("expected access token to be masked, got %#v", event.Payload["accessToken"])
		}
		nested, ok := event.Payload["nested"].(map[string]any)
		if !ok || nested["password"] != "[masked]" {
			t.Fatalf("expected nested password to be masked, got %#v", event.Payload["nested"])
		}
		if event.Payload["status"] != "failed" {
			t.Fatalf("expected non-sensitive field to remain, got %#v", event.Payload["status"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for stream event")
	}
}
