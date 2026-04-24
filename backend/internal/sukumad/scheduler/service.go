package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strconv"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	"basepro/backend/internal/logging"
	"basepro/backend/internal/sukumad/delivery"
	"basepro/backend/internal/sukumad/reporter"
	requests "basepro/backend/internal/sukumad/request"
	sukumadserver "basepro/backend/internal/sukumad/server"
	"github.com/jackc/pgx/v5/pgconn"
)

var schedulerCodePattern = regexp.MustCompile(`^[a-z0-9._-]+$`)

type Service struct {
	repo         Repository
	auditService *audit.Service
	clock        func() time.Time
	registry     *HandlerRegistry
}

func NewService(repository Repository, auditService ...*audit.Service) *Service {
	var auditSvc *audit.Service
	if len(auditService) > 0 {
		auditSvc = auditService[0]
	}
	service := &Service{
		repo:         repository,
		auditService: auditSvc,
		registry:     NewDefaultHandlerRegistry(),
		clock: func() time.Time {
			return time.Now().UTC()
		},
	}
	return service.WithDefaultMaintenanceHandlers()
}

func (s *Service) WithClock(clock func() time.Time) *Service {
	if clock != nil {
		s.clock = clock
	}
	return s
}

func (s *Service) WithRegistry(registry *HandlerRegistry) *Service {
	if registry != nil {
		s.registry = registry
	}
	return s
}

func (s *Service) WithDefaultMaintenanceHandlers() *Service {
	if s == nil {
		return s
	}
	handlers := newMaintenanceHandlers(maintenanceHandlerDependencies{
		repo: newMaintenanceRepository(s.repo),
		now:  s.clock,
	})
	for jobType, registration := range handlers {
		s.registry.Register(jobType, registration.handler)
	}
	return s
}

func (s *Service) WithIntegrationHandlers(deps integrationHandlerDependencies) *Service {
	if s == nil {
		return s
	}
	for jobType, registration := range newIntegrationHandlers(deps) {
		s.registry.Register(jobType, registration.handler)
	}
	return s
}

func (s *Service) WithIntegrationServices(
	serverLookup interface {
		GetServerByUID(context.Context, string) (sukumadserver.Record, error)
	},
	requestCreator interface {
		CreateExternalRequest(context.Context, requests.ExternalCreateInput) (requests.CreateResult, error)
	},
	submitter interface {
		Submit(context.Context, delivery.DispatchInput) (delivery.DispatchResult, error)
	},
) *Service {
	return s.WithIntegrationHandlers(integrationHandlerDependencies{
		serverLookup:   serverLookup,
		requestCreator: requestCreator,
		submitter:      submitter,
	})
}

func (s *Service) WithRapidProReporterSyncService(syncer interface {
	SyncUpdatedSince(context.Context, *time.Time, int, bool, bool) (reporter.SyncBatchResult, error)
}) *Service {
	if s == nil {
		return s
	}
	return s.WithIntegrationHandlers(integrationHandlerDependencies{
		reporterSyncer: syncer,
	})
}

func (s *Service) ListScheduledJobs(ctx context.Context, query ListQuery) (ListResult, error) {
	return s.repo.ListScheduledJobs(ctx, query)
}

func (s *Service) GetScheduledJob(ctx context.Context, id int64) (Record, error) {
	record, err := s.repo.GetScheduledJobByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"scheduled job not found"}})
		}
		return Record{}, err
	}
	return record, nil
}

func (s *Service) CreateScheduledJob(ctx context.Context, input CreateInput) (Record, error) {
	normalized, nextRunAt, details := s.normalizeInput(input)
	if len(details) > 0 {
		return Record{}, apperror.ValidationWithDetails("validation failed", details)
	}

	created, err := s.repo.CreateScheduledJob(ctx, CreateParams{
		UID:                 newUID(),
		Code:                normalized.Code,
		Name:                normalized.Name,
		Description:         normalized.Description,
		JobCategory:         normalized.JobCategory,
		JobType:             normalized.JobType,
		ScheduleType:        normalized.ScheduleType,
		ScheduleExpr:        normalized.ScheduleExpr,
		Timezone:            normalized.Timezone,
		Enabled:             normalized.Enabled,
		AllowConcurrentRuns: normalized.AllowConcurrentRuns,
		Config:              normalized.Config,
		NextRunAt:           nextRunAt,
	})
	if err != nil {
		if mapped := mapConstraintError(err); mapped != nil {
			return Record{}, mapped
		}
		return Record{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "scheduler.job.created",
		ActorUserID: input.ActorID,
		EntityType:  "scheduled_job",
		EntityID:    strPtr(fmt.Sprintf("%d", created.ID)),
		Metadata: map[string]any{
			"code":         created.Code,
			"name":         created.Name,
			"jobCategory":  created.JobCategory,
			"jobType":      created.JobType,
			"scheduleType": created.ScheduleType,
			"enabled":      created.Enabled,
		},
	})

	return created, nil
}

func (s *Service) UpdateScheduledJob(ctx context.Context, input UpdateInput) (Record, error) {
	if _, err := s.GetScheduledJob(ctx, input.ID); err != nil {
		return Record{}, err
	}

	normalized, nextRunAt, details := s.normalizeInput(CreateInput{
		Code:                input.Code,
		Name:                input.Name,
		Description:         input.Description,
		JobCategory:         input.JobCategory,
		JobType:             input.JobType,
		ScheduleType:        input.ScheduleType,
		ScheduleExpr:        input.ScheduleExpr,
		Timezone:            input.Timezone,
		Enabled:             input.Enabled,
		AllowConcurrentRuns: input.AllowConcurrentRuns,
		Config:              input.Config,
	})
	if len(details) > 0 {
		return Record{}, apperror.ValidationWithDetails("validation failed", details)
	}

	updated, err := s.repo.UpdateScheduledJob(ctx, UpdateParams{
		ID:                  input.ID,
		Code:                normalized.Code,
		Name:                normalized.Name,
		Description:         normalized.Description,
		JobCategory:         normalized.JobCategory,
		JobType:             normalized.JobType,
		ScheduleType:        normalized.ScheduleType,
		ScheduleExpr:        normalized.ScheduleExpr,
		Timezone:            normalized.Timezone,
		Enabled:             normalized.Enabled,
		AllowConcurrentRuns: normalized.AllowConcurrentRuns,
		Config:              normalized.Config,
		NextRunAt:           nextRunAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"scheduled job not found"}})
		}
		if mapped := mapConstraintError(err); mapped != nil {
			return Record{}, mapped
		}
		return Record{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "scheduler.job.updated",
		ActorUserID: input.ActorID,
		EntityType:  "scheduled_job",
		EntityID:    strPtr(fmt.Sprintf("%d", updated.ID)),
		Metadata: map[string]any{
			"code":         updated.Code,
			"name":         updated.Name,
			"jobCategory":  updated.JobCategory,
			"jobType":      updated.JobType,
			"scheduleType": updated.ScheduleType,
			"enabled":      updated.Enabled,
		},
	})

	return updated, nil
}

func (s *Service) SetScheduledJobEnabled(ctx context.Context, actorID *int64, id int64, enabled bool) (Record, error) {
	existing, err := s.GetScheduledJob(ctx, id)
	if err != nil {
		return Record{}, err
	}

	var nextRunAt *time.Time
	if enabled {
		if nextRunAt, err = s.CalculateNextRun(existing.ScheduleType, existing.ScheduleExpr, existing.Timezone, s.clock()); err != nil {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"scheduleExpr": []string{err.Error()}})
		}
	}

	record, err := s.repo.SetScheduledJobEnabled(ctx, SetEnabledParams{
		ID:        id,
		Enabled:   enabled,
		NextRunAt: nextRunAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Record{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"scheduled job not found"}})
		}
		return Record{}, err
	}

	action := "scheduler.job.disabled"
	if enabled {
		action = "scheduler.job.enabled"
	}
	s.logAudit(ctx, audit.Event{
		Action:      action,
		ActorUserID: actorID,
		EntityType:  "scheduled_job",
		EntityID:    strPtr(fmt.Sprintf("%d", record.ID)),
		Metadata: map[string]any{
			"code":    record.Code,
			"enabled": record.Enabled,
		},
	})

	return record, nil
}

func (s *Service) RunNow(ctx context.Context, actorID *int64, id int64) (RunRecord, error) {
	job, err := s.GetScheduledJob(ctx, id)
	if err != nil {
		return RunRecord{}, err
	}
	if !job.AllowConcurrentRuns {
		active, err := s.repo.HasActiveJobRuns(ctx, job.ID)
		if err != nil {
			return RunRecord{}, err
		}
		if active {
			return RunRecord{}, apperror.ValidationWithDetails("validation failed", map[string]any{
				"id": []string{"scheduled job already has an active run"},
			})
		}
	}

	now := s.clock()
	record, err := s.repo.CreateJobRun(ctx, CreateRunParams{
		UID:            newUID(),
		ScheduledJobID: job.ID,
		TriggerMode:    TriggerModeManual,
		ScheduledFor:   now,
		Status:         RunStatusPending,
		ResultSummary: map[string]any{
			"message": "Manual run queued by scheduler API",
		},
	})
	if err != nil {
		return RunRecord{}, err
	}

	s.logAudit(ctx, audit.Event{
		Action:      "scheduler.job.run_now",
		ActorUserID: actorID,
		EntityType:  "scheduled_job",
		EntityID:    strPtr(fmt.Sprintf("%d", job.ID)),
		Metadata: map[string]any{
			"code":      job.Code,
			"runId":     record.ID,
			"runUid":    record.UID,
			"trigger":   record.TriggerMode,
			"scheduled": record.ScheduledFor,
		},
	})

	return record, nil
}

func (s *Service) ListJobRuns(ctx context.Context, jobID int64, query RunListQuery) (RunListResult, error) {
	if _, err := s.GetScheduledJob(ctx, jobID); err != nil {
		return RunListResult{}, err
	}
	return s.repo.ListJobRuns(ctx, jobID, query)
}

func (s *Service) GetRun(ctx context.Context, id int64) (RunRecord, error) {
	record, err := s.repo.GetRunByID(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RunRecord{}, apperror.ValidationWithDetails("validation failed", map[string]any{"id": []string{"scheduled job run not found"}})
		}
		return RunRecord{}, err
	}
	return record, nil
}

func (s *Service) DispatchDueJobs(ctx context.Context, limit int) (DispatchCycleResult, error) {
	if limit <= 0 {
		limit = 1
	}
	now := s.clock()
	jobs, err := s.repo.ListDueScheduledJobs(ctx, now, limit)
	if err != nil {
		return DispatchCycleResult{}, err
	}

	result := DispatchCycleResult{
		CreatedRuns: make([]RunRecord, 0, len(jobs)),
		SkippedJobs: make([]int64, 0),
	}
	for _, job := range jobs {
		scheduledFor := now
		if job.NextRunAt != nil {
			scheduledFor = job.NextRunAt.UTC()
		}
		nextRunAt, err := s.CalculateNextFutureRun(job.ScheduleType, job.ScheduleExpr, job.Timezone, scheduledFor, now)
		if err != nil {
			return result, err
		}
		run, dispatched, err := s.repo.DispatchScheduledJob(ctx, DispatchJobParams{
			JobID:        job.ID,
			RunUID:       newUID(),
			ScheduledFor: scheduledFor,
			NextRunAt:    nextRunAt,
			TriggerMode:  TriggerModeScheduled,
			ResultSummary: map[string]any{
				"message": "Scheduled run queued by dispatcher",
			},
		})
		if err != nil {
			return result, err
		}
		if !dispatched {
			result.SkippedJobs = append(result.SkippedJobs, job.ID)
			continue
		}
		logging.ForContext(ctx).Info("scheduler_run_queued",
			s.schedulerLogAttrs(job, run, nil,
				slog.String("message", "Scheduled run queued by dispatcher"),
				slog.Time("next_run_at", valueOrZeroTime(nextRunAt)),
			)...,
		)
		result.CreatedRuns = append(result.CreatedRuns, run)
	}
	return result, nil
}

func (s *Service) RunPendingSchedulerRuns(ctx context.Context, workerID int64, batchSize int) error {
	if batchSize <= 0 {
		batchSize = 1
	}
	for index := 0; index < batchSize; index++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		run, err := s.repo.ClaimNextPendingRun(ctx, s.clock(), workerID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return err
		}
		job, err := s.repo.GetScheduledJobByID(ctx, run.ScheduledJobID)
		if err != nil {
			return s.finishRun(ctx, run.ID, RunStatusFailed, map[string]any{"message": "failed to load scheduled job"}, err)
		}
		logging.ForContext(ctx).Info("scheduler_run_claimed",
			s.schedulerLogAttrs(job, run, &workerID)...,
		)

		handler := s.registry.Lookup(job.JobType)
		if handler == nil {
			err := fmt.Errorf("unknown scheduler job type %q", job.JobType)
			logging.ForContext(ctx).Error("scheduler_run_failed",
				s.schedulerLogAttrs(job, run, &workerID,
					slog.String("status", RunStatusFailed),
					slog.String("error", err.Error()),
				)...,
			)
			if finishErr := s.finishRun(ctx, run.ID, RunStatusFailed, map[string]any{"jobType": job.JobType}, err); finishErr != nil {
				return finishErr
			}
			continue
		}

		exec := JobExecution{Job: job, Run: run, Now: s.clock()}
		logging.ForContext(ctx).Info("scheduler_run_started",
			s.schedulerLogAttrs(job, run, &workerID)...,
		)
		result, handlerErr := handler.Execute(ctx, exec)
		if handlerErr != nil {
			if errors.Is(handlerErr, context.Canceled) && ctx.Err() != nil {
				logging.ForContext(ctx).Warn("scheduler_run_cancelled",
					s.schedulerLogAttrs(job, run, &workerID,
						slog.String("status", RunStatusCancelled),
						slog.String("error", handlerErr.Error()),
					)...,
				)
				if finishErr := s.finishRun(ctx, run.ID, RunStatusCancelled, map[string]any{"jobType": job.JobType}, handlerErr); finishErr != nil {
					return finishErr
				}
				return ctx.Err()
			}
			logging.ForContext(ctx).Error("scheduler_run_failed",
				s.schedulerLogAttrs(job, run, &workerID,
					slog.String("status", RunStatusFailed),
					slog.String("error", handlerErr.Error()),
					slog.Any("result_summary", cloneJSONMap(result.ResultSummary)),
				)...,
			)
			if finishErr := s.finishRun(ctx, run.ID, RunStatusFailed, result.ResultSummary, handlerErr); finishErr != nil {
				return finishErr
			}
			continue
		}
		status := result.Status
		if status == "" {
			status = RunStatusSucceeded
		}
		logging.ForContext(ctx).Info("scheduler_run_finished",
			s.schedulerLogAttrs(job, run, &workerID,
				slog.String("status", status),
				slog.Any("result_summary", cloneJSONMap(result.ResultSummary)),
			)...,
		)
		if finishErr := s.finishRun(ctx, run.ID, status, result.ResultSummary, nil); finishErr != nil {
			return finishErr
		}
	}
	return nil
}

func (s *Service) finishRun(ctx context.Context, runID int64, status string, summary map[string]any, runErr error) error {
	finishedAt := s.clock()
	params := FinalizeRunParams{
		RunID:         runID,
		Status:        status,
		FinishedAt:    finishedAt,
		ErrorMessage:  errorString(runErr),
		ResultSummary: cloneJSONMap(summary),
		LastRunAt:     finishedAt,
	}
	switch status {
	case RunStatusSucceeded:
		params.LastSuccessAt = &finishedAt
	case RunStatusFailed, RunStatusCancelled:
		params.LastFailureAt = &finishedAt
	}
	_, err := s.repo.FinalizeJobRun(ctx, params)
	return err
}

func (s *Service) CalculateNextRun(scheduleType string, scheduleExpr string, timezone string, reference time.Time) (*time.Time, error) {
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		return nil, fmt.Errorf("timezone must be valid")
	}

	switch strings.ToLower(strings.TrimSpace(scheduleType)) {
	case ScheduleTypeInterval:
		duration, err := parseIntervalExpr(scheduleExpr)
		if err != nil {
			return nil, err
		}
		next := reference.In(location).Add(duration)
		nextUTC := next.UTC()
		return &nextUTC, nil
	case ScheduleTypeCron:
		next, err := nextCronTime(scheduleExpr, reference.In(location))
		if err != nil {
			return nil, err
		}
		nextUTC := next.UTC()
		return &nextUTC, nil
	default:
		return nil, fmt.Errorf("scheduleType must be one of cron or interval")
	}
}

func (s *Service) CalculateNextFutureRun(scheduleType string, scheduleExpr string, timezone string, anchor time.Time, after time.Time) (*time.Time, error) {
	location, err := time.LoadLocation(strings.TrimSpace(timezone))
	if err != nil {
		return nil, fmt.Errorf("timezone must be valid")
	}
	anchorLocal := anchor.In(location)
	afterLocal := after.In(location)

	switch strings.ToLower(strings.TrimSpace(scheduleType)) {
	case ScheduleTypeInterval:
		duration, err := parseIntervalExpr(scheduleExpr)
		if err != nil {
			return nil, err
		}
		next := anchorLocal.Add(duration)
		for !next.After(afterLocal) {
			next = next.Add(duration)
		}
		nextUTC := next.UTC()
		return &nextUTC, nil
	case ScheduleTypeCron:
		next, err := nextCronTime(scheduleExpr, anchorLocal)
		if err != nil {
			return nil, err
		}
		for !next.After(afterLocal) {
			next, err = nextCronTime(scheduleExpr, next)
			if err != nil {
				return nil, err
			}
		}
		nextUTC := next.UTC()
		return &nextUTC, nil
	default:
		return nil, fmt.Errorf("scheduleType must be one of cron or interval")
	}
}

func (s *Service) normalizeInput(input CreateInput) (CreateInput, *time.Time, map[string]any) {
	normalized := CreateInput{
		Code:                strings.ToLower(strings.TrimSpace(input.Code)),
		Name:                strings.TrimSpace(input.Name),
		Description:         strings.TrimSpace(input.Description),
		JobCategory:         strings.ToLower(strings.TrimSpace(input.JobCategory)),
		JobType:             strings.TrimSpace(input.JobType),
		ScheduleType:        strings.ToLower(strings.TrimSpace(input.ScheduleType)),
		ScheduleExpr:        strings.TrimSpace(input.ScheduleExpr),
		Timezone:            strings.TrimSpace(input.Timezone),
		Enabled:             input.Enabled,
		AllowConcurrentRuns: input.AllowConcurrentRuns,
		Config:              cloneJSONMap(input.Config),
	}
	if normalized.Timezone == "" {
		normalized.Timezone = "UTC"
	}

	details := map[string]any{}
	if normalized.Code == "" {
		details["code"] = []string{"is required"}
	} else if !schedulerCodePattern.MatchString(normalized.Code) {
		details["code"] = []string{"must contain only lowercase letters, numbers, dots, underscores, or hyphens"}
	}
	if normalized.Name == "" {
		details["name"] = []string{"is required"}
	}
	if normalized.JobCategory != JobCategoryIntegration && normalized.JobCategory != JobCategoryMaintenance {
		details["jobCategory"] = []string{"must be one of integration or maintenance"}
	}
	if normalized.JobType == "" {
		details["jobType"] = []string{"is required"}
	}
	if normalized.ScheduleType != ScheduleTypeCron && normalized.ScheduleType != ScheduleTypeInterval {
		details["scheduleType"] = []string{"must be one of cron or interval"}
	}
	if normalized.ScheduleExpr == "" {
		details["scheduleExpr"] = []string{"is required"}
	}
	if _, err := time.LoadLocation(normalized.Timezone); err != nil {
		details["timezone"] = []string{"must be a valid IANA timezone"}
	}
	if len(details) == 0 {
		if configDetails := s.registry.ValidateConfig(normalized.JobType, normalized.Config); len(configDetails) > 0 {
			for key, value := range configDetails {
				details[key] = value
			}
		}
	}

	var nextRunAt *time.Time
	if len(details) == 0 {
		next, err := s.CalculateNextRun(normalized.ScheduleType, normalized.ScheduleExpr, normalized.Timezone, s.clock())
		if err != nil {
			details["scheduleExpr"] = []string{err.Error()}
		} else if normalized.Enabled {
			nextRunAt = next
		}
	}

	return normalized, nextRunAt, details
}

func (s *Service) EnsureDefaultMaintenanceJobs(ctx context.Context) ([]Record, error) {
	defaults := []CreateInput{
		{
			Code:         "archive-old-requests",
			Name:         "Archive Old Requests",
			Description:  "Mark old terminal requests as archived for retention review.",
			JobCategory:  JobCategoryMaintenance,
			JobType:      "archive_old_requests",
			ScheduleType: ScheduleTypeCron,
			ScheduleExpr: "0 1 * * *",
			Timezone:     "UTC",
			Enabled:      true,
			Config: map[string]any{
				"dryRun":     false,
				"batchSize":  100,
				"maxAgeDays": 30,
			},
		},
		{
			Code:         "purge-old-logs",
			Name:         "Purge Old Logs",
			Description:  "Delete aged audit, request-event, poll, and worker log records.",
			JobCategory:  JobCategoryMaintenance,
			JobType:      "purge_old_logs",
			ScheduleType: ScheduleTypeCron,
			ScheduleExpr: "0 2 * * 0",
			Timezone:     "UTC",
			Enabled:      true,
			Config: map[string]any{
				"dryRun":     false,
				"batchSize":  500,
				"maxAgeDays": 30,
			},
		},
		{
			Code:         "mark-stuck-requests",
			Name:         "Mark Stuck Requests",
			Description:  "Block stale pending or processing requests that have stopped progressing.",
			JobCategory:  JobCategoryMaintenance,
			JobType:      "mark_stuck_requests",
			ScheduleType: ScheduleTypeInterval,
			ScheduleExpr: "10m",
			Timezone:     "UTC",
			Enabled:      true,
			Config: map[string]any{
				"dryRun":             false,
				"batchSize":          100,
				"staleCutoffMinutes": 30,
			},
		},
	}

	created := make([]Record, 0, len(defaults))
	for _, def := range defaults {
		_, err := s.repo.GetScheduledJobByCode(ctx, def.Code)
		if err == nil {
			continue
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		record, err := s.CreateScheduledJob(ctx, def)
		if err != nil {
			return nil, err
		}
		created = append(created, record)
	}
	return created, nil
}

func (s *Service) EnsureDefaultIntegrationJobs(ctx context.Context) ([]Record, error) {
	defaults := []CreateInput{
		{
			Code:         "rapidpro-reporter-sync",
			Name:         "RapidPro Reporter Sync",
			Description:  "Synchronize Rapidex reporters to RapidPro contacts incrementally.",
			JobCategory:  JobCategoryIntegration,
			JobType:      JobTypeRapidProReporterSync,
			ScheduleType: ScheduleTypeInterval,
			ScheduleExpr: "30m",
			Timezone:     "UTC",
			Enabled:      true,
			Config: map[string]any{
				"dryRun":     false,
				"batchSize":  100,
				"onlyActive": true,
			},
		},
	}

	created := make([]Record, 0, len(defaults))
	for _, def := range defaults {
		_, err := s.repo.GetScheduledJobByCode(ctx, def.Code)
		if err == nil {
			continue
		}
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			return nil, err
		}
		record, err := s.CreateScheduledJob(ctx, def)
		if err != nil {
			return nil, err
		}
		created = append(created, record)
	}
	return created, nil
}

func parseIntervalExpr(expr string) (time.Duration, error) {
	value := strings.ToLower(strings.TrimSpace(expr))
	if value == "" {
		return 0, fmt.Errorf("interval expression is required")
	}
	if duration, err := time.ParseDuration(value); err == nil {
		if duration <= 0 {
			return 0, fmt.Errorf("interval must be greater than zero")
		}
		return duration, nil
	}

	if strings.HasPrefix(value, "every ") {
		value = strings.TrimSpace(strings.TrimPrefix(value, "every "))
	}
	parts := strings.Fields(value)
	if len(parts) != 2 {
		return 0, fmt.Errorf("interval must use a Go duration like 15m or a simple form like '5 minutes'")
	}

	amount, err := strconv.Atoi(parts[0])
	if err != nil || amount <= 0 {
		return 0, fmt.Errorf("interval amount must be a positive integer")
	}

	unit := strings.TrimSuffix(parts[1], "s")
	switch unit {
	case "second":
		return time.Duration(amount) * time.Second, nil
	case "minute":
		return time.Duration(amount) * time.Minute, nil
	case "hour":
		return time.Duration(amount) * time.Hour, nil
	case "day":
		return time.Duration(amount) * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("interval unit must be seconds, minutes, hours, or days")
	}
}

type cronField struct {
	allowed map[int]struct{}
}

func newCronField() cronField {
	return cronField{allowed: make(map[int]struct{})}
}

func (f cronField) matches(value int) bool {
	_, ok := f.allowed[value]
	return ok
}

func nextCronTime(expr string, reference time.Time) (time.Time, error) {
	fields := strings.Fields(strings.TrimSpace(expr))
	if len(fields) != 5 {
		return time.Time{}, fmt.Errorf("cron must have 5 fields: minute hour day month weekday")
	}

	minutes, err := parseCronField(fields[0], 0, 59)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid minute field")
	}
	hours, err := parseCronField(fields[1], 0, 23)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid hour field")
	}
	days, err := parseCronField(fields[2], 1, 31)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid day field")
	}
	months, err := parseCronField(fields[3], 1, 12)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid month field")
	}
	weekdays, err := parseCronField(fields[4], 0, 6)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid weekday field")
	}

	candidate := reference.Truncate(time.Minute).Add(time.Minute)
	limit := candidate.Add(366 * 24 * time.Hour)
	for !candidate.After(limit) {
		if minutes.matches(candidate.Minute()) &&
			hours.matches(candidate.Hour()) &&
			days.matches(candidate.Day()) &&
			months.matches(int(candidate.Month())) &&
			weekdays.matches(int(candidate.Weekday())) {
			return candidate, nil
		}
		candidate = candidate.Add(time.Minute)
	}
	return time.Time{}, fmt.Errorf("unable to resolve next cron run within one year")
}

func parseCronField(expr string, min int, max int) (cronField, error) {
	field := newCronField()
	for _, part := range strings.Split(strings.TrimSpace(expr), ",") {
		if err := addCronPart(field.allowed, strings.TrimSpace(part), min, max); err != nil {
			return cronField{}, err
		}
	}
	if len(field.allowed) == 0 {
		return cronField{}, fmt.Errorf("empty field")
	}
	return field, nil
}

func addCronPart(target map[int]struct{}, expr string, min int, max int) error {
	if expr == "" {
		return fmt.Errorf("empty part")
	}
	if expr == "*" {
		for value := min; value <= max; value++ {
			target[value] = struct{}{}
		}
		return nil
	}

	step := 1
	base := expr
	if strings.Contains(expr, "/") {
		parts := strings.Split(expr, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid step")
		}
		base = parts[0]
		stepValue, err := strconv.Atoi(parts[1])
		if err != nil || stepValue <= 0 {
			return fmt.Errorf("invalid step")
		}
		step = stepValue
	}

	rangeStart := min
	rangeEnd := max
	switch {
	case base == "*" || base == "":
	case strings.Contains(base, "-"):
		parts := strings.Split(base, "-")
		if len(parts) != 2 {
			return fmt.Errorf("invalid range")
		}
		start, err := strconv.Atoi(parts[0])
		if err != nil {
			return err
		}
		end, err := strconv.Atoi(parts[1])
		if err != nil {
			return err
		}
		rangeStart = start
		rangeEnd = end
	default:
		value, err := strconv.Atoi(base)
		if err != nil {
			return err
		}
		rangeStart = value
		rangeEnd = value
	}

	if rangeStart < min || rangeEnd > max || rangeStart > rangeEnd {
		return fmt.Errorf("out of range")
	}
	for value := rangeStart; value <= rangeEnd; value += step {
		target[value] = struct{}{}
	}
	return nil
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func (s *Service) schedulerLogAttrs(job Record, run RunRecord, workerID *int64, extra ...slog.Attr) []any {
	attrs := []any{
		slog.Int64("job_id", job.ID),
		slog.String("job_uid", job.UID),
		slog.String("job_code", job.Code),
		slog.String("job_type", job.JobType),
		slog.Int64("run_id", run.ID),
		slog.String("run_uid", run.UID),
		slog.String("trigger_mode", run.TriggerMode),
		slog.Time("scheduled_for", run.ScheduledFor),
	}
	if workerID != nil {
		attrs = append(attrs, slog.Int64("worker_id", *workerID))
	}
	for _, attr := range extra {
		attrs = append(attrs, attr)
	}
	return attrs
}

func mapConstraintError(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return nil
	}

	if pgErr.Code != "23505" {
		return nil
	}

	switch pgErr.ConstraintName {
	case "scheduled_jobs_code_key":
		return apperror.ValidationWithDetails("validation failed", map[string]any{"code": []string{"must be unique"}})
	case "scheduled_jobs_uid_key", "scheduled_job_runs_uid_key":
		return apperror.ValidationWithDetails("validation failed", map[string]any{"uid": []string{"must be unique"}})
	default:
		return apperror.ValidationWithDetails("validation failed", map[string]any{"record": []string{"must be unique"}})
	}
}

func strPtr(value string) *string {
	return &value
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func valueOrZeroTime(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return value.UTC()
}
