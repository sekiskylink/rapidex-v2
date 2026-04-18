package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

type JobHandler interface {
	Execute(context.Context, JobExecution) (JobResult, error)
}

type HandlerRegistry struct {
	mu       sync.RWMutex
	handlers map[string]JobHandler
}

func NewHandlerRegistry() *HandlerRegistry {
	return &HandlerRegistry{handlers: make(map[string]JobHandler)}
}

func NewDefaultHandlerRegistry() *HandlerRegistry {
	registry := NewHandlerRegistry()
	registry.Register("metadata_sync", typedHandler[metadataSyncConfig](func(_ context.Context, exec JobExecution, cfg metadataSyncConfig) (JobResult, error) {
		return placeholderSucceeded(exec, "metadata_sync", cfg), nil
	}))
	registry.Register("export_pending_requests", typedHandler[exportPendingRequestsConfig](func(_ context.Context, exec JobExecution, cfg exportPendingRequestsConfig) (JobResult, error) {
		return placeholderSkipped(exec, "export_pending_requests", cfg, "export pending requests is not implemented yet"), nil
	}))
	registry.Register("reconciliation_pull", typedHandler[reconciliationPullConfig](func(_ context.Context, exec JobExecution, cfg reconciliationPullConfig) (JobResult, error) {
		return placeholderSkipped(exec, "reconciliation_pull", cfg, "reconciliation pull is not implemented yet"), nil
	}))
	registry.Register("scheduled_backfill", typedHandler[scheduledBackfillConfig](func(_ context.Context, exec JobExecution, cfg scheduledBackfillConfig) (JobResult, error) {
		return placeholderSkipped(exec, "scheduled_backfill", cfg, "scheduled backfill is not implemented yet"), nil
	}))
	registry.Register("archive_old_requests", typedHandler[archiveOldRequestsConfig](func(_ context.Context, exec JobExecution, cfg archiveOldRequestsConfig) (JobResult, error) {
		return placeholderSucceeded(exec, "archive_old_requests", cfg), nil
	}))
	registry.Register("purge_old_logs", typedHandler[purgeOldLogsConfig](func(_ context.Context, exec JobExecution, cfg purgeOldLogsConfig) (JobResult, error) {
		return placeholderSucceeded(exec, "purge_old_logs", cfg), nil
	}))
	registry.Register("mark_stuck_requests", typedHandler[markStuckRequestsConfig](func(_ context.Context, exec JobExecution, cfg markStuckRequestsConfig) (JobResult, error) {
		return placeholderSucceeded(exec, "mark_stuck_requests", cfg), nil
	}))
	registry.Register("cleanup_orphaned_records", typedHandler[cleanupOrphanedRecordsConfig](func(_ context.Context, exec JobExecution, cfg cleanupOrphanedRecordsConfig) (JobResult, error) {
		return placeholderSucceeded(exec, "cleanup_orphaned_records", cfg), nil
	}))
	return registry
}

func (r *HandlerRegistry) Register(jobType string, handler JobHandler) {
	if r == nil || handler == nil {
		return
	}
	key := strings.ToLower(strings.TrimSpace(jobType))
	if key == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.handlers[key] = handler
}

func (r *HandlerRegistry) Lookup(jobType string) JobHandler {
	if r == nil {
		return nil
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.handlers[strings.ToLower(strings.TrimSpace(jobType))]
}

type typedJobHandler[T any] struct {
	run func(context.Context, JobExecution, T) (JobResult, error)
}

func typedHandler[T any](run func(context.Context, JobExecution, T) (JobResult, error)) JobHandler {
	return typedJobHandler[T]{run: run}
}

func (h typedJobHandler[T]) Execute(ctx context.Context, exec JobExecution) (JobResult, error) {
	cfg, err := decodeTypedConfig[T](exec.Job.Config)
	if err != nil {
		return JobResult{}, err
	}
	return h.run(ctx, exec, cfg)
}

func decodeTypedConfig[T any](input map[string]any) (T, error) {
	var cfg T
	payload, err := json.Marshal(cloneJSONMap(input))
	if err != nil {
		return cfg, fmt.Errorf("marshal scheduler config: %w", err)
	}
	if err := json.Unmarshal(payload, &cfg); err != nil {
		return cfg, fmt.Errorf("decode scheduler config: %w", err)
	}
	return cfg, nil
}

type metadataSyncConfig struct {
	ServerCode string `json:"serverCode"`
	BatchSize  int    `json:"batchSize"`
	DryRun     bool   `json:"dryRun"`
}

type exportPendingRequestsConfig struct {
	ServerCode string `json:"serverCode"`
	Limit      int    `json:"limit"`
	DryRun     bool   `json:"dryRun"`
}

type reconciliationPullConfig struct {
	ServerCode string `json:"serverCode"`
	Since      string `json:"since"`
}

type scheduledBackfillConfig struct {
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	Limit     int    `json:"limit"`
	DryRun    bool   `json:"dryRun"`
}

type archiveOldRequestsConfig struct {
	OlderThanDays int  `json:"olderThanDays"`
	BatchSize     int  `json:"batchSize"`
	DryRun        bool `json:"dryRun"`
}

type purgeOldLogsConfig struct {
	OlderThanDays int  `json:"olderThanDays"`
	BatchSize     int  `json:"batchSize"`
	DryRun        bool `json:"dryRun"`
}

type markStuckRequestsConfig struct {
	OlderThanMinutes int  `json:"olderThanMinutes"`
	Limit            int  `json:"limit"`
	DryRun           bool `json:"dryRun"`
}

type cleanupOrphanedRecordsConfig struct {
	BatchSize int  `json:"batchSize"`
	DryRun    bool `json:"dryRun"`
}

func placeholderSucceeded(exec JobExecution, jobType string, cfg any) JobResult {
	return JobResult{
		Status: RunStatusSucceeded,
		ResultSummary: map[string]any{
			"jobType":      jobType,
			"runUid":       exec.Run.UID,
			"implemented":  false,
			"placeholder":  true,
			"message":      "Placeholder handler executed successfully",
			"acceptedConfig": cfg,
		},
	}
}

func placeholderSkipped(exec JobExecution, jobType string, cfg any, message string) JobResult {
	return JobResult{
		Status: RunStatusSkipped,
		ResultSummary: map[string]any{
			"jobType":        jobType,
			"runUid":         exec.Run.UID,
			"implemented":    false,
			"placeholder":    true,
			"message":        message,
			"acceptedConfig": cfg,
		},
	}
}
