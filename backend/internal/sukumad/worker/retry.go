package worker

import "context"

func NewRetryDefinition(service interface {
	RunRetryBatch(context.Context, Execution, int) error
}, batchSize int) Definition {
	if batchSize <= 0 {
		batchSize = 10
	}
	return Definition{
		Type: TypeRetry,
		Name: "retry-worker",
		Run: func(ctx context.Context, exec Execution) error {
			if service == nil {
				return nil
			}
			return service.RunRetryBatch(ctx, exec, batchSize)
		},
		Meta: map[string]any{
			"purpose":   "schedule delivery retries",
			"batchSize": batchSize,
		},
	}
}
