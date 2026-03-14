package devseed

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	asyncjobs "basepro/backend/internal/sukumad/async"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/server"
)

type Service struct {
	repo            *Repository
	serverService   *server.Service
	requestService  *request.Service
	deliveryService *delivery.Service
	asyncService    *asyncjobs.Service
}

func NewService(
	repo *Repository,
	serverService *server.Service,
	requestService *request.Service,
	deliveryService *delivery.Service,
	asyncService *asyncjobs.Service,
) *Service {
	return &Service{
		repo:            repo,
		serverService:   serverService,
		requestService:  requestService,
		deliveryService: deliveryService,
		asyncService:    asyncService,
	}
}

func (s *Service) Seed(ctx context.Context) (Summary, error) {
	if err := s.repo.CleanupSeedData(ctx, DemoSeedTag, DemoServerPrefix); err != nil {
		return Summary{}, err
	}

	trackerServer, err := s.serverService.CreateServer(ctx, server.CreateInput{
		Name:           "Demo DHIS2 Tracker",
		Code:           "demo-dhis2-tracker",
		SystemType:     "dhis2",
		BaseURL:        "https://play.im.dhis2.org/dev",
		EndpointType:   "http",
		HTTPMethod:     "POST",
		UseAsync:       false,
		ParseResponses: true,
		URLParams:      map[string]string{"importStrategy": "CREATE_AND_UPDATE"},
	})
	if err != nil {
		return Summary{}, fmt.Errorf("create tracker server: %w", err)
	}

	asyncServer, err := s.serverService.CreateServer(ctx, server.CreateInput{
		Name:           "Demo DHIS2 Async Tracker",
		Code:           "demo-dhis2-async",
		SystemType:     "dhis2",
		BaseURL:        "https://play.im.dhis2.org/dev",
		EndpointType:   "http",
		HTTPMethod:     "POST",
		UseAsync:       true,
		ParseResponses: true,
	})
	if err != nil {
		return Summary{}, fmt.Errorf("create async server: %w", err)
	}

	metadataServer, err := s.serverService.CreateServer(ctx, server.CreateInput{
		Name:           "Demo DHIS2 Metadata",
		Code:           "demo-dhis2-metadata",
		SystemType:     "dhis2",
		BaseURL:        "https://play.im.dhis2.org/dev",
		EndpointType:   "http",
		HTTPMethod:     "POST",
		UseAsync:       false,
		ParseResponses: true,
	})
	if err != nil {
		return Summary{}, fmt.Errorf("create metadata server: %w", err)
	}

	completedRequest, err := s.createRequest(ctx, request.CreateInput{
		SourceSystem:         "emr-demo",
		DestinationServerID:  trackerServer.ID,
		BatchID:              "demo-batch-001",
		CorrelationID:        "demo-completed-001",
		IdempotencyKey:       "demo-completed-001",
		URLSuffix:            "/api/tracker",
		DestinationServerIDs: []int64{metadataServer.ID},
		Payload:              rawJSON(`{"trackedEntity":"DemoPerson","orgUnit":"DiszpKrYNg8","program":"IpHINAT79UW","attributes":[{"attribute":"w75KJ2mc4zz","value":"Completed demo seed"}]}`),
		Extras: map[string]any{
			"seedTag":      DemoSeedTag,
			"scenario":     "completed-multi-target",
			"displayLabel": "Completed multi-target tracker import",
		},
	})
	if err != nil {
		return Summary{}, fmt.Errorf("create completed request: %w", err)
	}
	if err := s.markRequestCompleted(ctx, completedRequest); err != nil {
		return Summary{}, fmt.Errorf("complete seeded request: %w", err)
	}

	pendingRequest, err := s.createRequest(ctx, request.CreateInput{
		SourceSystem:         "lab-demo",
		DestinationServerID:  trackerServer.ID,
		DestinationServerIDs: []int64{metadataServer.ID},
		BatchID:              "demo-batch-002",
		CorrelationID:        "demo-pending-002",
		IdempotencyKey:       "demo-pending-002",
		URLSuffix:            "/api/dataValueSets",
		Payload:              rawJSON(`{"dataSet":"lyLU2wR22tC","completeDate":"2026-03-14","period":"202603","orgUnit":"DiszpKrYNg8","dataValues":[{"dataElement":"f7n9E0hX8qk","value":"17"}]}`),
		Extras: map[string]any{
			"seedTag":      DemoSeedTag,
			"scenario":     "pending-multi-target",
			"displayLabel": "Pending request with two targets",
		},
	})
	if err != nil {
		return Summary{}, fmt.Errorf("create pending request: %w", err)
	}

	blockedRequest, err := s.createRequest(ctx, request.CreateInput{
		SourceSystem:         "lab-demo",
		DestinationServerID:  trackerServer.ID,
		BatchID:              "demo-batch-003",
		CorrelationID:        "demo-blocked-003",
		IdempotencyKey:       "demo-blocked-003",
		URLSuffix:            "/api/dataValueSets",
		DependencyRequestIDs: []int64{pendingRequest.ID},
		Payload:              rawJSON(`{"dataSet":"lyLU2wR22tC","completeDate":"2026-03-14","period":"202603","orgUnit":"DiszpKrYNg8","dataValues":[{"dataElement":"f7n9E0hX8qk","value":"29"}]}`),
		Extras: map[string]any{
			"seedTag":      DemoSeedTag,
			"scenario":     "blocked-by-dependency",
			"displayLabel": "Blocked by dependency on pending request",
		},
	})
	if err != nil {
		return Summary{}, fmt.Errorf("create blocked request: %w", err)
	}

	failedRequest, err := s.createRequest(ctx, request.CreateInput{
		SourceSystem:        "hie-demo",
		DestinationServerID: trackerServer.ID,
		BatchID:             "demo-batch-004",
		CorrelationID:       "demo-failed-004",
		IdempotencyKey:      "demo-failed-004",
		URLSuffix:           "/api/metadata",
		Payload:             rawJSON(`{"metadata":{"organisationUnits":[{"id":"BadUnit","name":"Failed demo seed"}]}}`),
		Extras: map[string]any{
			"seedTag":      DemoSeedTag,
			"scenario":     "failed-terminal",
			"displayLabel": "Failed metadata push",
		},
	})
	if err != nil {
		return Summary{}, fmt.Errorf("create failed request: %w", err)
	}
	if err := s.markRequestFailed(ctx, failedRequest, 409, `{"status":"ERROR","description":"Demo conflict from seed"}`, "demo conflict from seed"); err != nil {
		return Summary{}, fmt.Errorf("fail seeded request: %w", err)
	}

	processingRequest, err := s.createRequest(ctx, request.CreateInput{
		SourceSystem:        "tracker-demo",
		DestinationServerID: asyncServer.ID,
		BatchID:             "demo-batch-005",
		CorrelationID:       "demo-processing-005",
		IdempotencyKey:      "demo-processing-005",
		URLSuffix:           "/api/tracker",
		Payload:             rawJSON(`{"trackedEntities":[{"trackedEntityType":"MCPQUTHX1Ze","orgUnit":"DiszpKrYNg8","attributes":[{"attribute":"w75KJ2mc4zz","value":"Async demo seed"}]}]}`),
		Extras: map[string]any{
			"seedTag":      DemoSeedTag,
			"scenario":     "processing-async",
			"displayLabel": "Processing async tracker import",
		},
	})
	if err != nil {
		return Summary{}, fmt.Errorf("create processing request: %w", err)
	}
	if err := s.markRequestProcessingAsync(ctx, processingRequest); err != nil {
		return Summary{}, fmt.Errorf("mark request processing async: %w", err)
	}

	_ = blockedRequest

	return Summary{
		SeedTag:        DemoSeedTag,
		ServersSeeded:  3,
		RequestsSeeded: 5,
	}, nil
}

func (s *Service) createRequest(ctx context.Context, input request.CreateInput) (request.Record, error) {
	created, err := s.requestService.CreateRequest(ctx, input)
	if err != nil {
		return request.Record{}, err
	}
	return s.requestService.GetRequest(ctx, created.ID)
}

func (s *Service) markRequestCompleted(ctx context.Context, record request.Record) error {
	for _, target := range record.Targets {
		deliveryID, err := latestDeliveryIDForServer(record, target.ServerID)
		if err != nil {
			return err
		}
		if _, err := s.deliveryService.MarkRunning(ctx, deliveryID); err != nil {
			return err
		}
		if _, err := s.deliveryService.MarkSucceeded(ctx, delivery.CompletionInput{
			ID:                  deliveryID,
			HTTPStatus:          intPtr(200),
			ResponseBody:        `{"status":"OK","description":"Demo seed completed successfully"}`,
			ResponseContentType: "application/json",
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) markRequestFailed(ctx context.Context, record request.Record, statusCode int, responseBody string, errorMessage string) error {
	deliveryID, err := latestDeliveryIDForServer(record, record.DestinationServerID)
	if err != nil {
		return err
	}
	if _, err := s.deliveryService.MarkRunning(ctx, deliveryID); err != nil {
		return err
	}
	if _, err := s.deliveryService.MarkFailed(ctx, delivery.CompletionInput{
		ID:           deliveryID,
		HTTPStatus:   intPtr(statusCode),
		ResponseBody: responseBody,
		ErrorMessage: errorMessage,
	}); err != nil {
		return err
	}
	return nil
}

func (s *Service) markRequestProcessingAsync(ctx context.Context, record request.Record) error {
	deliveryID, err := latestDeliveryIDForServer(record, record.DestinationServerID)
	if err != nil {
		return err
	}
	if _, err := s.deliveryService.MarkRunning(ctx, deliveryID); err != nil {
		return err
	}
	nextPollAt := time.Now().UTC().Add(10 * time.Minute)
	if _, err := s.asyncService.CreateTask(ctx, asyncjobs.CreateInput{
		DeliveryAttemptID: deliveryID,
		RemoteJobID:       "DEMO-ASYNC-JOB-005",
		PollURL:           "https://play.im.dhis2.org/dev/api/tracker/jobs/DEMO-ASYNC-JOB-005",
		RemoteStatus:      asyncjobs.StatePending,
		NextPollAt:        &nextPollAt,
		RemoteResponse: map[string]any{
			"seedTag": DemoSeedTag,
			"status":  "PENDING",
		},
	}); err != nil {
		return err
	}
	return nil
}

func latestDeliveryIDForServer(record request.Record, serverID int64) (int64, error) {
	for _, target := range record.Targets {
		if target.ServerID == serverID && target.LatestDeliveryID != nil {
			return *target.LatestDeliveryID, nil
		}
	}
	if record.DestinationServerID == serverID && record.LatestDeliveryID != nil {
		return *record.LatestDeliveryID, nil
	}
	return 0, fmt.Errorf("latest delivery not found for request %d server %d", record.ID, serverID)
}

func rawJSON(value string) json.RawMessage {
	return json.RawMessage(value)
}

func intPtr(value int) *int {
	return &value
}
