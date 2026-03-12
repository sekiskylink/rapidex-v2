package observability

import (
	"context"

	"basepro/backend/internal/sukumad/ratelimit"
	"basepro/backend/internal/sukumad/worker"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListWorkers(ctx context.Context, query worker.ListQuery) (worker.ListResult, error) {
	return s.repository.workers.ListRuns(ctx, query)
}

func (s *Service) GetWorker(ctx context.Context, id int64) (worker.Record, error) {
	return s.repository.workers.GetRun(ctx, id)
}

func (s *Service) ListRateLimits(ctx context.Context, query ratelimit.ListQuery) (ratelimit.ListResult, error) {
	return s.repository.rateLimits.ListPolicies(ctx, query)
}
