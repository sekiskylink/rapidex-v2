package worker

import (
	"context"

	asyncjobs "basepro/backend/internal/sukumad/async"
)

func NewPollDefinition(asyncService interface {
	PollDueTasks(context.Context, int, asyncjobs.RemotePoller) error
}, poller asyncjobs.RemotePoller, batchSize int) Definition {
	if batchSize <= 0 {
		batchSize = 10
	}
	return Definition{
		Type: TypePoll,
		Name: "poll-worker",
		Run: func(ctx context.Context, _ Execution) error {
			if asyncService == nil {
				return nil
			}
			return asyncService.PollDueTasks(ctx, batchSize, poller)
		},
		Meta: map[string]any{
			"purpose":   "poll async task state",
			"batchSize": batchSize,
		},
	}
}
