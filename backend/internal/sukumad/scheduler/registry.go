package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"
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
	registry.Register("metadata_sync", typedHandlerWithValidate[metadataSyncConfig](func(_ context.Context, exec JobExecution, cfg metadataSyncConfig) (JobResult, error) {
		return placeholderSucceeded(exec, "metadata_sync", cfg), nil
	}, nil))
	registry.Register("export_pending_requests", typedHandlerWithValidate[exportPendingRequestsConfig](func(_ context.Context, exec JobExecution, cfg exportPendingRequestsConfig) (JobResult, error) {
		return placeholderSkipped(exec, "export_pending_requests", cfg, "export pending requests is not implemented yet"), nil
	}, nil))
	registry.Register("reconciliation_pull", typedHandlerWithValidate[reconciliationPullConfig](func(_ context.Context, exec JobExecution, cfg reconciliationPullConfig) (JobResult, error) {
		return placeholderSkipped(exec, "reconciliation_pull", cfg, "reconciliation pull is not implemented yet"), nil
	}, nil))
	registry.Register("scheduled_backfill", typedHandlerWithValidate[scheduledBackfillConfig](func(_ context.Context, exec JobExecution, cfg scheduledBackfillConfig) (JobResult, error) {
		return placeholderSkipped(exec, "scheduled_backfill", cfg, "scheduled backfill is not implemented yet"), nil
	}, nil))
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

func (r *HandlerRegistry) ValidateConfig(jobType string, config map[string]any) map[string]any {
	handler := r.Lookup(jobType)
	validator, ok := handler.(jobConfigValidator)
	if !ok {
		return nil
	}
	return validator.ValidateConfig(config)
}

type typedJobHandler[T any] struct {
	run      func(context.Context, JobExecution, T) (JobResult, error)
	validate func(T) map[string]any
}

func typedHandler[T any](run func(context.Context, JobExecution, T) (JobResult, error)) JobHandler {
	return typedHandlerWithValidate(run, nil)
}

func typedHandlerWithValidate[T any](run func(context.Context, JobExecution, T) (JobResult, error), validate func(T) map[string]any) JobHandler {
	return typedJobHandler[T]{run: run, validate: validate}
}

func (h typedJobHandler[T]) Execute(ctx context.Context, exec JobExecution) (JobResult, error) {
	cfg, err := decodeTypedConfig[T](exec.Job.Config)
	if err != nil {
		return JobResult{}, err
	}
	return h.run(ctx, exec, cfg)
}

func (h typedJobHandler[T]) ValidateConfig(input map[string]any) map[string]any {
	cfg, err := decodeTypedConfig[T](input)
	if err != nil {
		return map[string]any{"config": []string{err.Error()}}
	}
	if h.validate == nil {
		return nil
	}
	return h.validate(cfg)
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
	MaxAgeDays int  `json:"maxAgeDays"`
	BatchSize  int  `json:"batchSize"`
	DryRun     bool `json:"dryRun"`
}

type purgeOldLogsConfig struct {
	MaxAgeDays int  `json:"maxAgeDays"`
	BatchSize  int  `json:"batchSize"`
	DryRun     bool `json:"dryRun"`
}

type markStuckRequestsConfig struct {
	StaleCutoffMinutes int  `json:"staleCutoffMinutes"`
	StaleCutoffHours   int  `json:"staleCutoffHours"`
	BatchSize          int  `json:"batchSize"`
	DryRun             bool `json:"dryRun"`
}

type cleanupOrphanedRecordsConfig struct {
	MaxAgeDays int  `json:"maxAgeDays"`
	BatchSize  int  `json:"batchSize"`
	DryRun     bool `json:"dryRun"`
}

func (c markStuckRequestsConfig) effectiveCutoffMinutes() int {
	if c.StaleCutoffMinutes > 0 {
		return c.StaleCutoffMinutes
	}
	if c.StaleCutoffHours > 0 {
		return c.StaleCutoffHours * 60
	}
	return 0
}

func (c markStuckRequestsConfig) staleCutoff() time.Duration {
	return time.Duration(c.effectiveCutoffMinutes()) * time.Minute
}

func placeholderSucceeded(exec JobExecution, jobType string, cfg any) JobResult {
	return JobResult{
		Status: RunStatusSucceeded,
		ResultSummary: map[string]any{
			"jobType":        jobType,
			"runUid":         exec.Run.UID,
			"implemented":    false,
			"placeholder":    true,
			"message":        "Placeholder handler executed successfully",
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
