package worker

import "context"

func NewIngestDefinition(run func(context.Context, Execution) error, batchSize int) Definition {
	if batchSize <= 0 {
		batchSize = 10
	}
	return Definition{
		Type: TypeIngest,
		Name: "directory-ingest-worker",
		Run:  run,
		Meta: map[string]any{
			"purpose":   "ingest directory request envelopes",
			"batchSize": batchSize,
		},
	}
}
