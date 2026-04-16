package request

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/delivery"
	sukumadserver "basepro/backend/internal/sukumad/server"
	"basepro/backend/internal/sukumad/traceevent"
)

type fakeRepo struct {
	listFn                      func(ctx context.Context, query ListQuery) (ListResult, error)
	getFn                       func(ctx context.Context, id int64) (Record, error)
	getByUIDFn                  func(ctx context.Context, uid string) (Record, error)
	listByBatchFn               func(ctx context.Context, batchID string) ([]Record, error)
	listByCorrelationFn         func(ctx context.Context, correlationID string) ([]Record, error)
	getBySourceAndIdempotencyFn func(ctx context.Context, sourceSystem string, idempotencyKey string) (Record, error)
	summaryFn                   func(ctx context.Context, query TargetStatusSummaryQuery) (TargetStatusSummary, error)
	createFn                    func(ctx context.Context, params CreateParams) (Record, error)
	updateFn                    func(ctx context.Context, id int64, status string, reason string, deferredUntil *time.Time) (Record, error)
	deleteFn                    func(ctx context.Context, id int64) error
}

func (f *fakeRepo) ListRequests(ctx context.Context, query ListQuery) (ListResult, error) {
	return f.listFn(ctx, query)
}

func (f *fakeRepo) GetRequestByID(ctx context.Context, id int64) (Record, error) {
	return f.getFn(ctx, id)
}

func (f *fakeRepo) GetRequestByUID(ctx context.Context, uid string) (Record, error) {
	if f.getByUIDFn == nil {
		return Record{}, sql.ErrNoRows
	}
	return f.getByUIDFn(ctx, uid)
}

func (f *fakeRepo) ListRequestsByBatchID(ctx context.Context, batchID string) ([]Record, error) {
	if f.listByBatchFn == nil {
		return []Record{}, nil
	}
	return f.listByBatchFn(ctx, batchID)
}

func (f *fakeRepo) ListRequestsByCorrelationID(ctx context.Context, correlationID string) ([]Record, error) {
	if f.listByCorrelationFn == nil {
		return []Record{}, nil
	}
	return f.listByCorrelationFn(ctx, correlationID)
}

func (f *fakeRepo) GetRequestBySourceSystemAndIdempotencyKey(ctx context.Context, sourceSystem string, idempotencyKey string) (Record, error) {
	if f.getBySourceAndIdempotencyFn == nil {
		return Record{}, sql.ErrNoRows
	}
	return f.getBySourceAndIdempotencyFn(ctx, sourceSystem, idempotencyKey)
}

func (f *fakeRepo) GetTargetStatusSummary(ctx context.Context, query TargetStatusSummaryQuery) (TargetStatusSummary, error) {
	if f.summaryFn == nil {
		return TargetStatusSummary{}, nil
	}
	return f.summaryFn(ctx, query)
}

func (f *fakeRepo) CreateRequest(ctx context.Context, params CreateParams) (Record, error) {
	return f.createFn(ctx, params)
}

func (f *fakeRepo) UpdateRequestStatus(ctx context.Context, id int64, status string, reason string, deferredUntil *time.Time) (Record, error) {
	if f.updateFn == nil {
		return Record{}, sql.ErrNoRows
	}
	return f.updateFn(ctx, id, status, reason, deferredUntil)
}

func (f *fakeRepo) DeleteRequest(ctx context.Context, id int64) error {
	if f.deleteFn == nil {
		return nil
	}
	return f.deleteFn(ctx, id)
}

func (f *fakeRepo) CreateTargets(context.Context, int64, []CreateTargetParams) ([]TargetRecord, error) {
	return []TargetRecord{}, nil
}

func (f *fakeRepo) ListTargetsByRequest(context.Context, int64) ([]TargetRecord, error) {
	return []TargetRecord{}, nil
}

func (f *fakeRepo) UpdateTarget(context.Context, UpdateTargetParams) (TargetRecord, error) {
	return TargetRecord{}, nil
}

func (f *fakeRepo) CreateDependencies(context.Context, int64, []int64) error {
	return nil
}

func (f *fakeRepo) ListDependencies(context.Context, int64) ([]DependencyRef, error) {
	return []DependencyRef{}, nil
}

func (f *fakeRepo) ListDependents(context.Context, int64) ([]DependencyRef, error) {
	return []DependencyRef{}, nil
}

func (f *fakeRepo) GetDependencyStatuses(context.Context, int64) ([]DependencyStatus, error) {
	return []DependencyStatus{}, nil
}

func (f *fakeRepo) DependencyPathExists(context.Context, int64, int64) (bool, error) {
	return false, nil
}

type fakeAuditRepo struct {
	events []audit.Event
}

type fakeEventWriter struct {
	events []traceevent.WriteInput
}

type fakeRequestDeliveryService struct {
	repo    *memoryRepository
	nextID  int64
	created []delivery.Record
}

type fakeServerResolver struct {
	items map[string]int64
}

func (f *fakeEventWriter) AppendEvent(_ context.Context, input traceevent.WriteInput) error {
	f.events = append(f.events, input)
	return nil
}

func (f *fakeAuditRepo) Insert(_ context.Context, event audit.Event) error {
	f.events = append(f.events, event)
	return nil
}

func (f *fakeAuditRepo) List(_ context.Context, _ audit.ListFilter) (audit.ListResult, error) {
	return audit.ListResult{}, nil
}

func (f *fakeRequestDeliveryService) CreatePendingDelivery(_ context.Context, input delivery.CreateInput) (delivery.Record, error) {
	f.nextID++
	record := delivery.Record{
		ID:        f.nextID,
		UID:       "del-test",
		RequestID: input.RequestID,
		ServerID:  input.ServerID,
		Status:    delivery.StatusPending,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	f.created = append(f.created, record)

	f.repo.mu.Lock()
	defer f.repo.mu.Unlock()
	item := f.repo.items[input.RequestID]
	for index := range item.Targets {
		if item.Targets[index].ServerID != input.ServerID {
			continue
		}
		item.Targets[index].LatestDeliveryID = &record.ID
		item.Targets[index].LatestDeliveryUID = record.UID
		item.Targets[index].LatestDeliveryStatus = record.Status
		break
	}
	f.repo.items[input.RequestID] = item
	return record, nil
}

func (f *fakeServerResolver) GetServerByUID(_ context.Context, uid string) (sukumadserver.Record, error) {
	id, ok := f.items[uid]
	if !ok {
		return sukumadserver.Record{}, sql.ErrNoRows
	}
	return sukumadserver.Record{ID: id, UID: uid, Name: "Server", Code: "server-code"}, nil
}

func TestServiceCreateRequestValidatesInput(t *testing.T) {
	service := NewService(&fakeRepo{}, audit.NewService(&fakeAuditRepo{}))

	_, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID: 0,
		Payload:             []byte(`not-json`),
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestServiceCreateRequestAcceptsTextQueryPayload(t *testing.T) {
	service := NewService(&fakeRepo{
		createFn: func(_ context.Context, params CreateParams) (Record, error) {
			return Record{
				ID:                12,
				UID:               params.UID,
				Status:            params.Status,
				PayloadBody:       params.PayloadBody,
				PayloadFormat:     params.PayloadFormat,
				SubmissionBinding: params.SubmissionBinding,
				Payload:           params.PayloadBody,
				CreatedAt:         time.Now().UTC(),
				UpdatedAt:         time.Now().UTC(),
			}, nil
		},
	}, audit.NewService(&fakeAuditRepo{}))

	record, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID: 3,
		Payload:             "trackedEntity=abc&orgUnit=ou-1",
		PayloadFormat:       PayloadFormatText,
		SubmissionBinding:   SubmissionBindingQuery,
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if record.PayloadFormat != PayloadFormatText || record.SubmissionBinding != SubmissionBindingQuery {
		t.Fatalf("expected text/query request contract, got %+v", record)
	}
	if record.PayloadBody != "trackedEntity=abc&orgUnit=ou-1" {
		t.Fatalf("unexpected payload body: %q", record.PayloadBody)
	}
}

func TestServiceCreateRequestDefaultsOptionalMetadataToEmptyStrings(t *testing.T) {
	service := NewService(&fakeRepo{
		createFn: func(_ context.Context, params CreateParams) (Record, error) {
			if params.BatchID != "" || params.CorrelationID != "" || params.IdempotencyKey != "" {
				t.Fatalf("expected empty optional metadata, got batch=%q correlation=%q idempotency=%q", params.BatchID, params.CorrelationID, params.IdempotencyKey)
			}
			return Record{
				ID:             13,
				UID:            params.UID,
				Status:         params.Status,
				BatchID:        params.BatchID,
				CorrelationID:  params.CorrelationID,
				IdempotencyKey: params.IdempotencyKey,
				PayloadBody:    params.PayloadBody,
				PayloadFormat:  params.PayloadFormat,
				Payload:        params.PayloadBody,
				CreatedAt:      time.Now().UTC(),
				UpdatedAt:      time.Now().UTC(),
			}, nil
		},
	}, audit.NewService(&fakeAuditRepo{}))

	record, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID: 3,
		BatchID:             "   ",
		CorrelationID:       "\t",
		IdempotencyKey:      " ",
		Payload:             []byte(`{"trackedEntity":"123"}`),
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if record.BatchID != "" || record.CorrelationID != "" || record.IdempotencyKey != "" {
		t.Fatalf("expected empty optional metadata on record, got %+v", record)
	}
}

func TestServiceListRequestsProjectsConfiguredMetadataColumns(t *testing.T) {
	service := NewService(&fakeRepo{
		listFn: func(_ context.Context, query ListQuery) (ListResult, error) {
			return ListResult{
				Items: []Record{
					{
						ID:     1,
						UID:    "req-1",
						Extras: map[string]any{"patientId": "P-100", "retries": 2, "submittedAt": "2026-04-11T10:00:00Z"},
					},
				},
				Total:    1,
				Page:     query.Page,
				PageSize: query.PageSize,
			}, nil
		},
	})

	result, err := service.ListRequests(context.Background(), ListQuery{
		Page:     1,
		PageSize: 25,
		MetadataColumns: []MetadataColumn{
			{Key: "patientId", Label: "Patient ID", Type: MetadataColumnTypeString, Searchable: true, VisibleByDefault: true},
			{Key: "retries", Label: "Retries", Type: MetadataColumnTypeNumber, Searchable: false, VisibleByDefault: true},
			{Key: "submittedAt", Label: "Submitted", Type: MetadataColumnTypeDateTime, Searchable: false, VisibleByDefault: false},
		},
	})
	if err != nil {
		t.Fatalf("list requests: %v", err)
	}
	if len(result.MetadataColumns) != 3 {
		t.Fatalf("expected projected metadata columns, got %+v", result.MetadataColumns)
	}
	if got := result.Items[0].ProjectedMetadata["patientId"]; got != "P-100" {
		t.Fatalf("expected patientId projection, got %#v", got)
	}
	if got := result.Items[0].ProjectedMetadata["retries"]; got != int64(2) && got != 2 && got != float64(2) {
		t.Fatalf("expected numeric projection, got %#v", got)
	}
	if got := result.Items[0].ProjectedMetadata["submittedAt"]; got != "2026-04-11T10:00:00Z" {
		t.Fatalf("expected datetime projection, got %#v", got)
	}
}

func TestServiceCreateRequestRejectsJSONQueryArrayPayload(t *testing.T) {
	service := NewService(&fakeRepo{}, audit.NewService(&fakeAuditRepo{}))

	_, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID: 3,
		Payload:             []byte(`[1,2,3]`),
		PayloadFormat:       PayloadFormatJSON,
		SubmissionBinding:   SubmissionBindingQuery,
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
}

func TestServiceCreateRequestWritesAuditEvent(t *testing.T) {
	auditRepo := &fakeAuditRepo{}
	eventWriter := &fakeEventWriter{}
	service := NewService(&fakeRepo{
		createFn: func(_ context.Context, params CreateParams) (Record, error) {
			return Record{
				ID:                    11,
				UID:                   params.UID,
				DestinationServerID:   params.DestinationServerID,
				DestinationServerName: "DHIS2 Uganda",
				Status:                params.Status,
				CorrelationID:         params.CorrelationID,
				PayloadBody:           params.PayloadBody,
				Payload:               []byte(params.PayloadBody),
				CreatedAt:             time.Now().UTC(),
				UpdatedAt:             time.Now().UTC(),
				CreatedBy:             params.CreatedBy,
			}, nil
		},
	}, audit.NewService(auditRepo)).WithEventWriter(eventWriter)

	actorID := int64(8)
	created, err := service.CreateRequest(context.Background(), CreateInput{
		SourceSystem:        "emr",
		DestinationServerID: 3,
		CorrelationID:       "corr-1",
		Payload:             []byte(`{"trackedEntity":"123"}`),
		Extras:              map[string]any{"priority": "high"},
		ActorID:             &actorID,
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}
	if created.Status != StatusPending {
		t.Fatalf("expected pending status, got %s", created.Status)
	}
	if len(auditRepo.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(auditRepo.events))
	}
	if auditRepo.events[0].Action != "request.created" {
		t.Fatalf("expected request.created, got %s", auditRepo.events[0].Action)
	}
	if len(eventWriter.events) != 1 || eventWriter.events[0].EventType != traceevent.EventRequestCreated {
		t.Fatalf("expected request.created event write, got %+v", eventWriter.events)
	}
}

func TestServiceGetRequestNotFound(t *testing.T) {
	service := NewService(&fakeRepo{
		getFn: func(_ context.Context, _ int64) (Record, error) {
			return Record{}, sql.ErrNoRows
		},
	}, audit.NewService(&fakeAuditRepo{}))

	if _, err := service.GetRequest(context.Background(), 99); err == nil {
		t.Fatal("expected not found error")
	}
}

func TestServiceDeleteRequestWritesAuditWithoutBodies(t *testing.T) {
	auditRepo := &fakeAuditRepo{}
	deleted := int64(0)
	service := NewService(&fakeRepo{
		getFn: func(_ context.Context, id int64) (Record, error) {
			return Record{
				ID:                    id,
				UID:                   "req-uid",
				Status:                StatusProcessing,
				DestinationServerID:   3,
				DestinationServerName: "DHIS2 Uganda",
				CorrelationID:         "corr-1",
				PayloadBody:           `{"secret":"hidden"}`,
			}, nil
		},
		deleteFn: func(_ context.Context, id int64) error {
			deleted = id
			return nil
		},
	}, audit.NewService(auditRepo))

	actorID := int64(8)
	if err := service.DeleteRequest(context.Background(), &actorID, 44); err != nil {
		t.Fatalf("delete request: %v", err)
	}
	if deleted != 44 {
		t.Fatalf("expected delete id 44, got %d", deleted)
	}
	if len(auditRepo.events) != 1 || auditRepo.events[0].Action != "request.deleted" {
		t.Fatalf("expected request.deleted audit event, got %+v", auditRepo.events)
	}
	if _, ok := auditRepo.events[0].Metadata["payloadBody"]; ok {
		t.Fatalf("delete audit metadata must not include payload body: %+v", auditRepo.events[0].Metadata)
	}
	if _, ok := auditRepo.events[0].Metadata["responseBody"]; ok {
		t.Fatalf("delete audit metadata must not include response body: %+v", auditRepo.events[0].Metadata)
	}
}

func TestServiceCreateExternalRequestResolvesUIDs(t *testing.T) {
	repo := NewRepository().(*memoryRepository)
	deliverySvc := &fakeRequestDeliveryService{repo: repo}
	service := NewService(repo, audit.NewService(&fakeAuditRepo{})).
		WithDeliveryService(deliverySvc).
		WithServerService(&fakeServerResolver{items: map[string]int64{
			"srv-primary": 3,
			"srv-copy":    4,
		}})

	result, err := service.CreateExternalRequest(context.Background(), ExternalCreateInput{
		SourceSystem:          "emr",
		DestinationServerUID:  "srv-primary",
		DestinationServerUIDs: []string{"srv-copy"},
		BatchID:               "batch-1",
		CorrelationID:         "corr-1",
		IdempotencyKey:        "idem-1",
		Payload:               []byte(`{"trackedEntity":"123"}`),
	})
	if err != nil {
		t.Fatalf("create external request: %v", err)
	}
	if !result.Created || result.Deduped {
		t.Fatalf("expected newly created request result, got %+v", result)
	}
	if result.Record.SourceSystem != "emr" || result.Record.BatchID != "batch-1" || result.Record.CorrelationID != "corr-1" {
		t.Fatalf("unexpected created record %+v", result.Record)
	}
	if len(result.Record.Targets) != 2 {
		t.Fatalf("expected two targets from uid resolution, got %+v", result.Record.Targets)
	}
}

func TestServiceCreateExternalRequestReturnsExistingByIdempotencyKey(t *testing.T) {
	repo := NewRepository().(*memoryRepository)
	service := NewService(repo, audit.NewService(&fakeAuditRepo{})).
		WithServerService(&fakeServerResolver{items: map[string]int64{"srv-primary": 3}})

	first, err := service.CreateExternalRequest(context.Background(), ExternalCreateInput{
		SourceSystem:         "emr",
		DestinationServerUID: "srv-primary",
		IdempotencyKey:       "idem-1",
		Payload:              []byte(`{"trackedEntity":"123"}`),
	})
	if err != nil {
		t.Fatalf("create first external request: %v", err)
	}
	second, err := service.CreateExternalRequest(context.Background(), ExternalCreateInput{
		SourceSystem:         "emr",
		DestinationServerUID: "srv-primary",
		IdempotencyKey:       "idem-1",
		Payload:              []byte(`{"trackedEntity":"123"}`),
	})
	if err != nil {
		t.Fatalf("replay external request: %v", err)
	}
	if !second.Deduped || second.Created {
		t.Fatalf("expected deduped replay, got %+v", second)
	}
	if second.Record.UID != first.Record.UID {
		t.Fatalf("expected same request uid on replay, got first=%s second=%s", first.Record.UID, second.Record.UID)
	}
}

func TestServiceGetExternalSummaryReturnsServerScopedCounts(t *testing.T) {
	service := NewService(&fakeRepo{
		summaryFn: func(_ context.Context, query TargetStatusSummaryQuery) (TargetStatusSummary, error) {
			if query.ServerID != 3 {
				t.Fatalf("expected server id 3, got %+v", query)
			}
			return TargetStatusSummary{
				Total:      12,
				Pending:    2,
				Blocked:    1,
				Processing: 3,
				Succeeded:  5,
				Failed:     1,
			}, nil
		},
	}, audit.NewService(&fakeAuditRepo{})).
		WithServerService(&fakeServerResolver{items: map[string]int64{"srv-primary": 3}})

	startDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2026, 4, 15, 23, 59, 59, 0, time.UTC)
	summary, err := service.GetExternalSummary(context.Background(), "srv-primary", startDate, endDate)
	if err != nil {
		t.Fatalf("get external summary: %v", err)
	}
	if summary.DestinationServer.UID != "srv-primary" || summary.Summary.Total != 12 {
		t.Fatalf("unexpected summary payload %+v", summary)
	}
	if summary.Period.StartDate != "2026-04-01" || summary.Period.EndDate != "2026-04-15" {
		t.Fatalf("unexpected period %+v", summary.Period)
	}
}

func TestServiceGetExternalSummaryReturnsValidationForUnknownServer(t *testing.T) {
	service := NewService(&fakeRepo{}, audit.NewService(&fakeAuditRepo{})).
		WithServerService(&fakeServerResolver{items: map[string]int64{}})

	if _, err := service.GetExternalSummary(context.Background(), "missing", time.Now().UTC(), time.Now().UTC()); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestServiceReleasesDependentRequestsWhenDependencyCompletes(t *testing.T) {
	repo := NewRepository().(*memoryRepository)
	eventWriter := &fakeEventWriter{}
	deliverySvc := &fakeRequestDeliveryService{repo: repo}
	service := NewService(repo, audit.NewService(&fakeAuditRepo{})).
		WithEventWriter(eventWriter).
		WithDeliveryService(deliverySvc)

	dependency, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID: 3,
		Payload:             []byte(`{"trackedEntity":"dep"}`),
	})
	if err != nil {
		t.Fatalf("create dependency request: %v", err)
	}
	dependent, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID:  3,
		DependencyRequestIDs: []int64{dependency.ID},
		Payload:              []byte(`{"trackedEntity":"child"}`),
	})
	if err != nil {
		t.Fatalf("create dependent request: %v", err)
	}
	if dependent.Status != StatusBlocked {
		t.Fatalf("expected dependent request to be blocked, got %s", dependent.Status)
	}

	if err := service.SetTargetSucceeded(context.Background(), dependency.ID, 3); err != nil {
		t.Fatalf("complete dependency target: %v", err)
	}

	reloaded, err := service.GetRequest(context.Background(), dependent.ID)
	if err != nil {
		t.Fatalf("reload dependent request: %v", err)
	}
	if reloaded.Status == StatusBlocked {
		t.Fatalf("expected dependent request to be released, got blocked with reason %q", reloaded.StatusReason)
	}
	if len(reloaded.Targets) != 1 || reloaded.Targets[0].Status != TargetStatusPending {
		t.Fatalf("expected released target to return to pending, got %+v", reloaded.Targets)
	}
	if reloaded.Targets[0].LastReleasedAt == nil {
		t.Fatalf("expected release timestamp to be recorded, got %+v", reloaded.Targets[0])
	}
	if len(deliverySvc.created) != 2 {
		t.Fatalf("expected only the initial durable deliveries, got %d", len(deliverySvc.created))
	}
	if reloaded.Status != StatusPending {
		t.Fatalf("expected dependent request to return to pending for worker pickup, got %s", reloaded.Status)
	}

	found := false
	for _, event := range eventWriter.events {
		if event.EventType == "request.unblocked.dependency" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected request.unblocked.dependency event, got %+v", eventWriter.events)
	}
}

func TestServiceCreateRequestPersistsDurableStateWithoutInlineSubmission(t *testing.T) {
	repo := NewRepository().(*memoryRepository)
	deliverySvc := &fakeRequestDeliveryService{repo: repo}
	service := NewService(repo, audit.NewService(&fakeAuditRepo{})).WithDeliveryService(deliverySvc)

	created, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID:  3,
		DestinationServerIDs: []int64{4},
		DependencyRequestIDs: []int64{9},
		Payload:              []byte(`{"trackedEntity":"child"}`),
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	if created.Status != StatusBlocked {
		t.Fatalf("expected blocked request while dependency is incomplete, got %s", created.Status)
	}
	if len(created.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %+v", created.Targets)
	}
	for _, target := range created.Targets {
		if target.Status != TargetStatusBlocked {
			t.Fatalf("expected target to remain blocked durably, got %+v", target)
		}
		if target.LatestDeliveryID == nil || target.LatestDeliveryStatus != delivery.StatusPending {
			t.Fatalf("expected pending durable delivery on target, got %+v", target)
		}
	}
	if len(created.Dependencies) != 1 || created.Dependencies[0].DependsOnRequestID != 9 {
		t.Fatalf("expected dependency link to persist, got %+v", created.Dependencies)
	}
	if len(deliverySvc.created) != 2 {
		t.Fatalf("expected one pending delivery per target, got %d", len(deliverySvc.created))
	}
}

func TestServiceCreateRequestLeavesUnblockedRequestPendingForWorkerPickup(t *testing.T) {
	repo := NewRepository().(*memoryRepository)
	deliverySvc := &fakeRequestDeliveryService{repo: repo}
	service := NewService(repo, audit.NewService(&fakeAuditRepo{})).WithDeliveryService(deliverySvc)

	created, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID: 3,
		Payload:             []byte(`{"trackedEntity":"child"}`),
	})
	if err != nil {
		t.Fatalf("create request: %v", err)
	}

	if created.Status != StatusPending {
		t.Fatalf("expected request to remain pending for worker pickup, got %s", created.Status)
	}
	if len(created.Targets) != 1 {
		t.Fatalf("expected one target, got %+v", created.Targets)
	}
	target := created.Targets[0]
	if target.Status != TargetStatusPending {
		t.Fatalf("expected pending target, got %+v", target)
	}
	if target.LatestDeliveryID == nil || target.LatestDeliveryStatus != delivery.StatusPending {
		t.Fatalf("expected pending delivery metadata on target, got %+v", target)
	}
	if len(deliverySvc.created) != 1 {
		t.Fatalf("expected one durable pending delivery, got %d", len(deliverySvc.created))
	}
}

func TestServiceFailsDependentsWhenDependencyFails(t *testing.T) {
	repo := NewRepository().(*memoryRepository)
	eventWriter := &fakeEventWriter{}
	deliverySvc := &fakeRequestDeliveryService{repo: repo}
	service := NewService(repo, audit.NewService(&fakeAuditRepo{})).
		WithEventWriter(eventWriter).
		WithDeliveryService(deliverySvc)

	dependency, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID: 3,
		Payload:             []byte(`{"trackedEntity":"dep"}`),
	})
	if err != nil {
		t.Fatalf("create dependency request: %v", err)
	}
	dependent, err := service.CreateRequest(context.Background(), CreateInput{
		DestinationServerID:  3,
		DependencyRequestIDs: []int64{dependency.ID},
		Payload:              []byte(`{"trackedEntity":"child"}`),
	})
	if err != nil {
		t.Fatalf("create dependent request: %v", err)
	}
	if err := service.SetTargetFailed(context.Background(), dependency.ID, 3, "upstream failed"); err != nil {
		t.Fatalf("fail dependency target: %v", err)
	}

	reloaded, err := service.GetRequest(context.Background(), dependent.ID)
	if err != nil {
		t.Fatalf("reload dependent request: %v", err)
	}
	if reloaded.Status != StatusFailed || reloaded.StatusReason != "dependency_failed" {
		t.Fatalf("expected dependent request failure from dependency, got status=%s reason=%s", reloaded.Status, reloaded.StatusReason)
	}
	if len(reloaded.Targets) != 1 || reloaded.Targets[0].Status != TargetStatusFailed || reloaded.Targets[0].BlockedReason != "dependency_failed" {
		t.Fatalf("expected dependent target failure, got %+v", reloaded.Targets)
	}

	found := false
	for _, event := range eventWriter.events {
		if event.EventType == "request.failed.dependency" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected request.failed.dependency event, got %+v", eventWriter.events)
	}
}
