package worker

import "context"

func NewSendDefinition(service interface {
	RunSendBatch(context.Context, Execution, int) error
}, batchSize int) Definition {
	if batchSize <= 0 {
		batchSize = 10
	}
	return Definition{
		Type: TypeSend,
		Name: "send-worker",
		Run: func(ctx context.Context, exec Execution) error {
			if service == nil {
				return nil
			}
			return service.RunSendBatch(ctx, exec, batchSize)
		},
		Meta: map[string]any{
			"purpose":   "dispatch delivery attempts",
			"batchSize": batchSize,
		},
	}
}
