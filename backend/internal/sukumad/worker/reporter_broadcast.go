package worker

import (
	"context"
	"time"
)

func NewReporterBroadcastDefinition(service interface {
	RunQueuedBroadcasts(context.Context, int64, func(string, int), int, time.Duration) error
}, batchSize int, claimTimeout time.Duration) Definition {
	if batchSize <= 0 {
		batchSize = 10
	}
	if claimTimeout <= 0 {
		claimTimeout = time.Minute
	}
	return Definition{
		Type: TypeReporterBroadcast,
		Name: "reporter-broadcast-worker",
		Run: func(ctx context.Context, exec Execution) error {
			if service == nil {
				return nil
			}
			return service.RunQueuedBroadcasts(ctx, exec.RunID, exec.AddCount, batchSize, claimTimeout)
		},
		Meta: map[string]any{
			"purpose":      "dispatch queued reporter broadcasts",
			"batchSize":    batchSize,
			"claimTimeout": claimTimeout.String(),
		},
	}
}
