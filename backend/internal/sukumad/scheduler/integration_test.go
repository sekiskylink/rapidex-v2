package scheduler

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"basepro/backend/internal/logging"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/reporter"
	requests "basepro/backend/internal/sukumad/request"
	sukumadserver "basepro/backend/internal/sukumad/server"
)

type fakeSchedulerServerLookup struct {
	record sukumadserver.Record
	err    error
}

func (f fakeSchedulerServerLookup) GetServerByUID(context.Context, string) (sukumadserver.Record, error) {
	return f.record, f.err
}

type fakeSchedulerSubmitter struct {
	input  delivery.DispatchInput
	result delivery.DispatchResult
	err    error
}

func (f *fakeSchedulerSubmitter) Submit(_ context.Context, input delivery.DispatchInput) (delivery.DispatchResult, error) {
	f.input = input
	return f.result, f.err
}

type fakeSchedulerRequestCreator struct {
	input  requests.ExternalCreateInput
	result requests.CreateResult
	err    error
}

type fakeSchedulerReporterSyncer struct {
	result reporter.SyncBatchResult
	err    error
}

func testIntPtr(value int) *int {
	return &value
}

func (f *fakeSchedulerRequestCreator) CreateExternalRequest(_ context.Context, input requests.ExternalCreateInput) (requests.CreateResult, error) {
	f.input = input
	return f.result, f.err
}

func (f fakeSchedulerReporterSyncer) SyncUpdatedSince(context.Context, *time.Time, int, bool, bool) (reporter.SyncBatchResult, error) {
	return f.result, f.err
}

func TestCreateScheduledJobRejectsInvalidURLCallConfig(t *testing.T) {
	svc := NewService(NewRepository()).WithIntegrationHandlers(integrationHandlerDependencies{})

	_, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "url-call",
		Name:         "URL Call",
		JobCategory:  JobCategoryIntegration,
		JobType:      JobTypeURLCall,
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "15m",
		Timezone:     "UTC",
		Enabled:      true,
		Config: map[string]any{
			"payloadFormat": "json",
			"payload":       map[string]any{"ping": true},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestURLCallSchedulerJobDispatchesThroughIntegrationServer(t *testing.T) {
	now := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	submitter := &fakeSchedulerSubmitter{
		result: delivery.DispatchResult{
			HTTPStatus:           testIntPtr(200),
			ResponseContentType:  "application/json",
			ResponseBodyFiltered: true,
			ResponseSummary:      map[string]any{"bytes": float64(42)},
			Terminal:             true,
			Succeeded:            true,
		},
	}
	svc := NewService(NewRepository()).
		WithClock(func() time.Time { return now }).
		WithIntegrationHandlers(integrationHandlerDependencies{
			serverLookup: fakeSchedulerServerLookup{record: sukumadserver.Record{
				ID:                      7,
				UID:                     "srv-uid",
				Code:                    "dhis2",
				Name:                    "DHIS2",
				SystemType:              "dhis2",
				BaseURL:                 "https://dhis.example.test",
				HTTPMethod:              "POST",
				ResponseBodyPersistence: "filter",
				Headers:                 map[string]string{"X-Test": "true"},
				URLParams:               map[string]string{"mode": "sync"},
			}},
			submitter: submitter,
		})

	job, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "url-call",
		Name:         "URL Call",
		JobCategory:  JobCategoryIntegration,
		JobType:      JobTypeURLCall,
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "15m",
		Timezone:     "UTC",
		Enabled:      true,
		Config: map[string]any{
			"destinationServerUid":    "srv-uid",
			"urlSuffix":               "/api/ping",
			"payloadFormat":           "json",
			"submissionBinding":       "body",
			"responseBodyPersistence": "discard",
			"payload":                 map[string]any{"ping": true},
		},
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := svc.RunNow(context.Background(), nil, job.ID); err != nil {
		t.Fatalf("run now: %v", err)
	}
	if err := svc.RunPendingSchedulerRuns(context.Background(), 99, 1); err != nil {
		t.Fatalf("run pending scheduler runs: %v", err)
	}

	if submitter.input.Server.Code != "dhis2" || submitter.input.URLSuffix != "/api/ping" {
		t.Fatalf("unexpected dispatch input: %+v", submitter.input)
	}
	if submitter.input.PayloadBody != `{"ping":true}` {
		t.Fatalf("unexpected payload body: %s", submitter.input.PayloadBody)
	}
	runs, err := svc.ListJobRuns(context.Background(), job.ID, RunListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs.Items) != 1 || runs.Items[0].Status != RunStatusSucceeded {
		t.Fatalf("expected succeeded run, got %+v", runs.Items)
	}
	if runs.Items[0].ResultSummary["destinationServerCode"] != "dhis2" {
		t.Fatalf("expected server code in summary, got %+v", runs.Items[0].ResultSummary)
	}
}

func TestRequestExchangeSchedulerJobCreatesExternalRequest(t *testing.T) {
	now := time.Date(2026, 4, 20, 9, 0, 0, 0, time.UTC)
	creator := &fakeSchedulerRequestCreator{
		result: requests.CreateResult{
			Created: true,
			Record: requests.Record{
				ID:                    42,
				UID:                   "req-uid",
				Status:                requests.StatusPending,
				CorrelationID:         "scheduler:request-exchange:run-uid",
				DestinationServerUID:  "srv-uid",
				DestinationServerCode: "dhis2",
				Targets:               []requests.TargetRecord{{ID: 100}},
			},
		},
	}
	svc := NewService(NewRepository()).
		WithClock(func() time.Time { return now }).
		WithIntegrationHandlers(integrationHandlerDependencies{requestCreator: creator})

	job, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "request-exchange",
		Name:         "Request Exchange",
		JobCategory:  JobCategoryIntegration,
		JobType:      JobTypeRequestExchange,
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "15m",
		Timezone:     "UTC",
		Enabled:      true,
		Config: map[string]any{
			"sourceSystem":         "scheduler",
			"destinationServerUid": "srv-uid",
			"destinationServerUids": []any{
				"srv-cc",
			},
			"idempotencyKeyPrefix": "scheduled",
			"payloadFormat":        "json",
			"submissionBinding":    "body",
			"payload":              map[string]any{"event": "daily"},
			"metadata":             map[string]any{"owner": "scheduler"},
		},
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	run, err := svc.RunNow(context.Background(), nil, job.ID)
	if err != nil {
		t.Fatalf("run now: %v", err)
	}
	if err := svc.RunPendingSchedulerRuns(context.Background(), 99, 1); err != nil {
		t.Fatalf("run pending scheduler runs: %v", err)
	}

	if creator.input.SourceSystem != "scheduler" || creator.input.DestinationServerUID != "srv-uid" {
		t.Fatalf("unexpected create input: %+v", creator.input)
	}
	if creator.input.CorrelationID != "scheduler:request-exchange:"+run.UID {
		t.Fatalf("unexpected correlation id: %s", creator.input.CorrelationID)
	}
	if creator.input.IdempotencyKey != "scheduled:"+run.UID {
		t.Fatalf("unexpected idempotency key: %s", creator.input.IdempotencyKey)
	}
	runs, err := svc.ListJobRuns(context.Background(), job.ID, RunListQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list runs: %v", err)
	}
	if len(runs.Items) != 1 || runs.Items[0].ResultSummary["requestUid"] != "req-uid" {
		t.Fatalf("expected request summary, got %+v", runs.Items)
	}
}

func TestRapidProReporterSyncSchedulerJobEmitsBatchLogs(t *testing.T) {
	now := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	logOutput := captureSchedulerIntegrationLogs(t)
	syncer := fakeSchedulerReporterSyncer{
		result: reporter.SyncBatchResult{
			Scanned:    3,
			Synced:     2,
			Created:    1,
			Updated:    1,
			Failed:     1,
			DryRun:     false,
			OnlyActive: true,
			WatermarkTo: func() *time.Time {
				value := now.Add(time.Minute)
				return &value
			}(),
		},
		err: errors.New("1 reporter syncs failed: remote validation"),
	}
	svc := NewService(NewRepository()).
		WithClock(func() time.Time { return now }).
		WithIntegrationHandlers(integrationHandlerDependencies{reporterSyncer: syncer})

	job, err := svc.CreateScheduledJob(context.Background(), CreateInput{
		Code:         "rapidpro-sync",
		Name:         "RapidPro Sync",
		JobCategory:  JobCategoryIntegration,
		JobType:      JobTypeRapidProReporterSync,
		ScheduleType: ScheduleTypeInterval,
		ScheduleExpr: "15m",
		Timezone:     "UTC",
		Enabled:      true,
		Config: map[string]any{
			"batchSize":  50,
			"dryRun":     false,
			"onlyActive": true,
		},
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := svc.RunNow(context.Background(), nil, job.ID); err != nil {
		t.Fatalf("run now: %v", err)
	}
	if err := svc.RunPendingSchedulerRuns(context.Background(), 101, 1); err != nil {
		t.Fatalf("run pending scheduler runs: %v", err)
	}

	assertSchedulerIntegrationLogContains(t, logOutput.String(),
		"rapidpro_reporter_sync_batch_started",
		"\"job_code\":\"rapidpro-sync\"",
		"\"batch_size\":50",
		"rapidpro_reporter_sync_batch_failed",
		"\"failed_count\":1",
		"\"error\":\"1 reporter syncs failed: remote validation\"",
	)
}

func captureSchedulerIntegrationLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var logOutput bytes.Buffer
	logging.SetOutput(&logOutput)
	logging.ApplyConfig(logging.Config{Level: "info", Format: "json"})
	t.Cleanup(func() {
		logging.SetOutput(nil)
		logging.ApplyConfig(logging.Config{Level: "info", Format: "console"})
	})
	return &logOutput
}

func assertSchedulerIntegrationLogContains(t *testing.T, logs string, fragments ...string) {
	t.Helper()
	for _, fragment := range fragments {
		if !strings.Contains(logs, fragment) {
			t.Fatalf("expected logs to contain %q, got:\n%s", fragment, logs)
		}
	}
}
