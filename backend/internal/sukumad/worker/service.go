package worker

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
)

type Service struct {
	repo         Repository
	auditService *audit.Service
}

func NewService(repository Repository, auditService ...*audit.Service) *Service {
	var auditSvc *audit.Service
	if len(auditService) > 0 {
		auditSvc = auditService[0]
	}
	return &Service{repo: repository, auditService: auditSvc}
}

func (s *Service) ListRuns(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.ListRuns(ctx, query)
}

func (s *Service) GetRun(ctx context.Context, id int64) (Record, error) {
	record, err := s.repo.GetRunByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"worker not found"}})
		}
		return Record{}, err
	}
	return record, nil
}

func (s *Service) StartRun(ctx context.Context, def Definition) (Record, error) {
	now := time.Now().UTC()
	record, err := s.repo.CreateRun(ctx, CreateParams{
		UID:        newUID(),
		WorkerType: def.Type,
		WorkerName: def.Name,
		Status:     StatusStarting,
		StartedAt:  now,
		Meta:       cloneJSONMap(def.Meta),
	})
	if err != nil {
		return Record{}, err
	}
	s.logAudit(ctx, audit.Event{
		Action:     "worker.started",
		EntityType: "worker_run",
		EntityID:   strPtr(fmt.Sprintf("%d", record.ID)),
		Metadata: map[string]any{
			"uid":        record.UID,
			"workerType": record.WorkerType,
			"workerName": record.WorkerName,
		},
	})
	return s.UpdateStatus(ctx, record.ID, StatusRunning, nil, record.Meta)
}

func (s *Service) Heartbeat(ctx context.Context, id int64, meta map[string]any) (Record, error) {
	current, err := s.GetRun(ctx, id)
	if err != nil {
		return Record{}, err
	}
	now := time.Now().UTC()
	return s.repo.UpdateRun(ctx, UpdateParams{
		ID:              id,
		Status:          current.Status,
		LastHeartbeatAt: &now,
		StoppedAt:       current.StoppedAt,
		Meta:            mergeMeta(current.Meta, meta),
	})
}

func (s *Service) UpdateStatus(ctx context.Context, id int64, status string, stoppedAt *time.Time, meta map[string]any) (Record, error) {
	current, err := s.GetRun(ctx, id)
	if err != nil {
		return Record{}, err
	}
	updated, err := s.repo.UpdateRun(ctx, UpdateParams{
		ID:              id,
		Status:          status,
		StoppedAt:       cloneTimePtr(stoppedAt),
		LastHeartbeatAt: current.LastHeartbeatAt,
		Meta:            mergeMeta(current.Meta, meta),
	})
	if err != nil {
		return Record{}, err
	}
	if status == StatusStopped {
		s.logAudit(ctx, audit.Event{
			Action:     "worker.stopped",
			EntityType: "worker_run",
			EntityID:   strPtr(fmt.Sprintf("%d", updated.ID)),
			Metadata: map[string]any{
				"uid":        updated.UID,
				"workerType": updated.WorkerType,
				"workerName": updated.WorkerName,
			},
		})
	}
	return updated, nil
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func mergeMeta(current map[string]any, next map[string]any) map[string]any {
	merged := cloneJSONMap(current)
	for key, value := range next {
		merged[key] = value
	}
	return merged
}

type Manager struct {
	service *Service
	defs    []Definition
}

func NewManager(service *Service, defs ...Definition) *Manager {
	return &Manager{service: service, defs: defs}
}

func (m *Manager) Start(ctx context.Context) <-chan error {
	errCh := make(chan error, len(m.defs))
	var wg sync.WaitGroup

	for _, def := range m.defs {
		definition := def
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := m.runWorker(ctx, definition); err != nil && !errors.Is(err, context.Canceled) {
				errCh <- err
			}
		}()
	}

	go func() {
		wg.Wait()
		close(errCh)
	}()

	return errCh
}

func (m *Manager) runWorker(ctx context.Context, def Definition) error {
	if m.service == nil {
		return nil
	}
	if def.Interval <= 0 {
		def.Interval = 250 * time.Millisecond
	}
	if def.HeartbeatInterval <= 0 {
		def.HeartbeatInterval = 100 * time.Millisecond
	}
	if def.Run == nil {
		def.Run = func(context.Context, Execution) error { return nil }
	}

	record, err := m.service.StartRun(ctx, def)
	if err != nil {
		return err
	}

	heartbeatTicker := time.NewTicker(def.HeartbeatInterval)
	defer heartbeatTicker.Stop()
	workTicker := time.NewTicker(def.Interval)
	defer workTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			now := time.Now().UTC()
			_, stopErr := m.service.UpdateStatus(context.Background(), record.ID, StatusStopped, &now, map[string]any{"shutdown": "context cancelled"})
			if stopErr != nil {
				return stopErr
			}
			return ctx.Err()
		case <-heartbeatTicker.C:
			if _, err := m.service.Heartbeat(ctx, record.ID, nil); err != nil {
				return err
			}
		case <-workTicker.C:
			if err := def.Run(ctx, Execution{RunID: record.ID}); err != nil {
				now := time.Now().UTC()
				_, updateErr := m.service.UpdateStatus(context.Background(), record.ID, StatusFailed, &now, map[string]any{"lastError": err.Error()})
				if updateErr != nil {
					return updateErr
				}
				return err
			}
		}
	}
}

func strPtr(v string) *string {
	return &v
}
