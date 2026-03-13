package retention

import (
	"context"
	"fmt"
	"time"

	"basepro/backend/internal/audit"
	"basepro/backend/internal/sukumad/traceevent"
)

type Service struct {
	repo         Repository
	auditService *audit.Service
	eventWriter  traceevent.Writer
}

func NewService(repository Repository, auditService ...*audit.Service) *Service {
	var auditSvc *audit.Service
	if len(auditService) > 0 {
		auditSvc = auditService[0]
	}
	return &Service{repo: repository, auditService: auditSvc}
}

func (s *Service) WithEventWriter(eventWriter traceevent.Writer) *Service {
	s.eventWriter = eventWriter
	return s
}

func (s *Service) Run(ctx context.Context, input RunInput) (RunResult, error) {
	batchSize := input.BatchSize
	if batchSize <= 0 {
		batchSize = 100
	}
	cutoff := input.Cutoff.UTC()

	s.appendEvent(ctx, traceevent.WriteInput{
		EventType:       "retention.run.started",
		EventLevel:      "info",
		Message:         traceevent.Message("Retention run started", "Retention run started for cutoff %s", cutoff.Format(time.RFC3339)),
		Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
		SourceComponent: "retention.service",
		EventData:       map[string]any{"cutoff": cutoff, "batchSize": batchSize, "dryRun": input.DryRun},
	})

	candidates, err := s.repo.ListPurgeCandidates(ctx, cutoff, batchSize)
	if err != nil {
		s.appendEvent(ctx, traceevent.WriteInput{
			EventType:       "retention.run.failed",
			EventLevel:      "error",
			Message:         traceevent.Message("Retention run failed", "Retention run failed: %v", err),
			Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
			SourceComponent: "retention.service",
			EventData:       map[string]any{"cutoff": cutoff, "batchSize": batchSize, "dryRun": input.DryRun, "error": err.Error()},
		})
		return RunResult{}, err
	}

	result := RunResult{
		Cutoff:            cutoff,
		BatchSize:         batchSize,
		DryRun:            input.DryRun,
		Scanned:           len(candidates),
		CandidateRequests: append([]Candidate{}, candidates...),
	}

	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return RunResult{}, err
		}
		if input.DryRun {
			result.Counts.Requests++
			continue
		}

		counts, err := s.repo.PurgeRequest(ctx, candidate.RequestID)
		if err != nil {
			s.appendEvent(ctx, traceevent.WriteInput{
				RequestID:       &candidate.RequestID,
				EventType:       "retention.run.failed",
				EventLevel:      "error",
				Message:         traceevent.Message("Retention request purge failed", "Retention purge failed for request %s", candidate.RequestUID),
				Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
				SourceComponent: "retention.service",
				EventData:       map[string]any{"requestUid": candidate.RequestUID, "error": err.Error()},
			})
			return RunResult{}, err
		}
		mergeCounts(&result.Counts, counts)
		s.appendEvent(ctx, traceevent.WriteInput{
			RequestID:       &candidate.RequestID,
			EventType:       "retention.request.purged",
			EventLevel:      "info",
			Message:         traceevent.Message("Retention request purged", "Retention purged request %s", candidate.RequestUID),
			Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
			SourceComponent: "retention.service",
			EventData: map[string]any{
				"requestUid": candidate.RequestUID,
				"counts":     counts,
			},
		})
	}

	s.logAudit(ctx, audit.Event{
		Action:     "retention.run",
		EntityType: "sukumad_retention",
		Metadata: map[string]any{
			"cutoff":    cutoff.Format(time.RFC3339),
			"batchSize": batchSize,
			"dryRun":    input.DryRun,
			"scanned":   result.Scanned,
			"counts":    result.Counts,
		},
	})
	s.appendEvent(ctx, traceevent.WriteInput{
		EventType:       "retention.run.completed",
		EventLevel:      "info",
		Message:         traceevent.Message("Retention run completed", "Retention run completed; scanned=%d dryRun=%t", result.Scanned, result.DryRun),
		Actor:           traceevent.Actor{Type: traceevent.ActorSystem},
		SourceComponent: "retention.service",
		EventData: map[string]any{
			"cutoff":    cutoff,
			"batchSize": batchSize,
			"dryRun":    input.DryRun,
			"scanned":   result.Scanned,
			"counts":    result.Counts,
		},
	})

	return result, nil
}

func mergeCounts(into *PurgeCounts, next PurgeCounts) {
	into.AsyncTaskPolls += next.AsyncTaskPolls
	into.AsyncTasks += next.AsyncTasks
	into.RequestEvents += next.RequestEvents
	into.DeliveryAttempts += next.DeliveryAttempts
	into.RequestTargets += next.RequestTargets
	into.Dependencies += next.Dependencies
	into.Requests += next.Requests
}

func (s *Service) appendEvent(ctx context.Context, input traceevent.WriteInput) {
	if s.eventWriter == nil {
		return
	}
	_ = s.eventWriter.AppendEvent(ctx, input)
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func (s *Service) RunFromConfig(ctx context.Context, cutoff time.Time, batchSize int, dryRun bool) error {
	_, err := s.Run(ctx, RunInput{
		Cutoff:    cutoff,
		BatchSize: batchSize,
		DryRun:    dryRun,
	})
	if err != nil {
		return fmt.Errorf("run retention: %w", err)
	}
	return nil
}
