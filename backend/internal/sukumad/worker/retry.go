package worker

import "context"

func NewRetryDefinition(run func(context.Context, Execution) error) Definition {
	return Definition{
		Type: TypeRetry,
		Name: "retry-worker",
		Run:  run,
		Meta: map[string]any{"purpose": "schedule delivery retries"},
	}
}
