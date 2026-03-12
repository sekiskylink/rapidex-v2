package worker

import "context"

func NewSendDefinition(run func(context.Context, Execution) error) Definition {
	return Definition{
		Type: TypeSend,
		Name: "send-worker",
		Run:  run,
		Meta: map[string]any{"purpose": "dispatch delivery attempts"},
	}
}
