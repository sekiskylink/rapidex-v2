package observability

import (
	"context"

	"basepro/backend/internal/sukumad/ratelimit"
	"basepro/backend/internal/sukumad/worker"
)

type Repository struct {
	workers interface {
		ListRuns(context.Context, worker.ListQuery) (worker.ListResult, error)
		GetRun(context.Context, int64) (worker.Record, error)
	}
	rateLimits interface {
		ListPolicies(context.Context, ratelimit.ListQuery) (ratelimit.ListResult, error)
	}
}

func NewRepository(
	workers interface {
		ListRuns(context.Context, worker.ListQuery) (worker.ListResult, error)
		GetRun(context.Context, int64) (worker.Record, error)
	},
	rateLimits interface {
		ListPolicies(context.Context, ratelimit.ListQuery) (ratelimit.ListResult, error)
	},
) *Repository {
	return &Repository{workers: workers, rateLimits: rateLimits}
}
