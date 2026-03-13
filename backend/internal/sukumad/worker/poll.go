package worker

import (
	"context"
	"time"

	asyncjobs "basepro/backend/internal/sukumad/async"
)

func NewPollDefinition(asyncService interface {
	PollDueTasks(context.Context, asyncjobs.PollExecution, int, asyncjobs.RemotePoller) error
}, poller asyncjobs.RemotePoller, batchSize int, claimTimeout time.Duration) Definition {
	if batchSize <= 0 {
		batchSize = 10
	}
	if claimTimeout <= 0 {
		claimTimeout = time.Minute
	}
	return Definition{
		Type: TypePoll,
		Name: "poll-worker",
		Run: func(ctx context.Context, exec Execution) error {
			if asyncService == nil {
				return nil
			}
			return asyncService.PollDueTasks(ctx, asyncjobs.PollExecution{
				WorkerRunID:  exec.RunID,
				ClaimTimeout: claimTimeout,
				Observe:      exec.AddCount,
			}, batchSize, poller)
		},
		Meta: map[string]any{
			"purpose":      "poll async task state",
			"batchSize":    batchSize,
			"claimTimeout": claimTimeout.String(),
		},
	}
}
