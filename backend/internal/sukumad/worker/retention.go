package worker

import (
	"context"
	"time"
)

func NewRetentionDefinition(service interface {
	RunFromConfig(context.Context, time.Time, int, bool) error
}, enabled bool, cutoff func() time.Time, batchSize int, dryRun bool) Definition {
	if batchSize <= 0 {
		batchSize = 100
	}
	return Definition{
		Type: TypeRetention,
		Name: "retention-worker",
		Run: func(ctx context.Context, _ Execution) error {
			if !enabled || service == nil {
				return nil
			}
			return service.RunFromConfig(ctx, cutoff(), batchSize, dryRun)
		},
		Meta: map[string]any{
			"purpose":   "purge terminal Sukumad request history",
			"batchSize": batchSize,
			"dryRun":    dryRun,
		},
	}
}
