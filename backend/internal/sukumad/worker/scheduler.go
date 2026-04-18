package worker

import "context"

func NewSchedulerDefinition(service interface {
	RunPendingSchedulerRuns(context.Context, int64, int) error
}, batchSize int) Definition {
	if batchSize <= 0 {
		batchSize = 1
	}
	return Definition{
		Type: TypeSchedulerRun,
		Name: "scheduler-run-worker",
		Run: func(ctx context.Context, exec Execution) error {
			if service == nil {
				return nil
			}
			return service.RunPendingSchedulerRuns(ctx, exec.RunID, batchSize)
		},
		Meta: map[string]any{
			"purpose":   "execute scheduled job runs",
			"batchSize": batchSize,
		},
	}
}
