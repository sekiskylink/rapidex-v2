package rapidex

import (
    "context"
    "errors"
    "fmt"
)

// IntegrationService coordinates the processing of RapidPro webhooks.  It
// fetches the appropriate mapping configuration, transforms the webhook
// payload into a DHIS2 aggregate payload, and passes it to the Sukumad
// request engine.  Dependencies on reporter and org unit services are
// optional and can be nil; if provided they can be used to resolve
// organisation units via reporter assignments.
type IntegrationService struct {
    mappingManager *Manager
    // reporterSvc may be nil if reporter lookup is not available yet.
    // It is expected to implement a GetByContactUUID(ctx, uuid string) method.
    reporterSvc interface{}
    // requestSvc may be nil; it should implement a method to create an
    // exchange_request given an aggregate payload.
    requestSvc interface{}
}

// NewIntegrationService constructs a new service.  Passing nil for
// reporterSvc or requestSvc is allowed but will limit functionality.
func NewIntegrationService(mgr *Manager, reporterSvc interface{}, requestSvc interface{}) *IntegrationService {
    return &IntegrationService{
        mappingManager: mgr,
        reporterSvc:    reporterSvc,
        requestSvc:     requestSvc,
    }
}

// ProcessWebhook processes a RapidPro webhook event.  It looks up the
// mapping configuration by flow UUID, transforms the event into an
// AggregatePayload, and (if configured) forwards the payload to the
// request service.  If no mapping exists, an error is returned.
func (s *IntegrationService) ProcessWebhook(ctx context.Context, webhook RapidProWebhook) error {
    cfg, ok := s.mappingManager.Get(webhook.FlowUUID)
    if !ok {
        return fmt.Errorf("no mapping configured for flow %s", webhook.FlowUUID)
    }
    payload, err := MapToAggregate(webhook, cfg)
    if err != nil {
        return fmt.Errorf("failed to map webhook: %w", err)
    }
    // TODO: if reporterSvc != nil, look up reporter by contact UUID or
    // phone number and override payload.OrgUnit accordingly.
    if s.requestSvc == nil {
        // For now, simply log the payload; in production this should
        // create a Sukumad exchange_request via requestSvc.
        fmt.Printf("Would enqueue aggregate payload: %+v\n", payload)
        return nil
    }
    // Attempt to call the request service; use reflection to call a
    // Create method if available.  Reflection allows decoupling from
    // concrete implementations.
    // Note: this is a placeholder until the full integration is
    // implemented.
    return errors.New("request service integration not implemented")
}