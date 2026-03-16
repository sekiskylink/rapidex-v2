package ingest

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"basepro/backend/internal/apperror"
	"basepro/backend/internal/audit"
	requests "basepro/backend/internal/sukumad/request"
	"basepro/backend/internal/sukumad/traceevent"
)

type Service struct {
	repo         Repository
	requestSvc   requestCreator
	auditService *audit.Service
	eventWriter  traceevent.Writer
	now          func() time.Time
	readFile     func(string) ([]byte, error)
	statFile     func(string) (fs.FileInfo, error)
	readDir      func(string) ([]os.DirEntry, error)
	mkdirAll     func(string, os.FileMode) error
	rename       func(string, string) error
}

func NewService(repository Repository, requestSvc requestCreator, auditService ...*audit.Service) *Service {
	var auditSvc *audit.Service
	if len(auditService) > 0 {
		auditSvc = auditService[0]
	}
	return &Service{
		repo:         repository,
		requestSvc:   requestSvc,
		auditService: auditSvc,
		now:          func() time.Time { return time.Now().UTC() },
		readFile:     os.ReadFile,
		statFile:     os.Stat,
		readDir:      os.ReadDir,
		mkdirAll:     os.MkdirAll,
		rename:       os.Rename,
	}
}

func (s *Service) WithEventWriter(eventWriter traceevent.Writer) *Service {
	s.eventWriter = eventWriter
	return s
}

func (s *Service) DiscoverDirectory(ctx context.Context, cfg RuntimeConfig) (int, error) {
	entries, err := s.readDir(cfg.InboxPath)
	if err != nil {
		return 0, fmt.Errorf("read ingest inbox: %w", err)
	}
	discovered := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		record, err := s.DiscoverPath(ctx, filepath.Join(cfg.InboxPath, entry.Name()), cfg)
		if err != nil {
			return discovered, err
		}
		if record.ID > 0 {
			discovered++
		}
	}
	return discovered, nil
}

func (s *Service) DiscoverPath(ctx context.Context, path string, cfg RuntimeConfig) (Record, error) {
	cleanPath := filepath.Clean(path)
	if !strings.HasPrefix(cleanPath, filepath.Clean(cfg.InboxPath)+string(os.PathSeparator)) && cleanPath != filepath.Clean(cfg.InboxPath) {
		return Record{}, nil
	}
	if !extensionAllowed(cleanPath, cfg.AllowedExtensions) {
		return Record{}, nil
	}
	info, err := s.statFile(cleanPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Record{}, nil
		}
		return Record{}, fmt.Errorf("stat ingest file: %w", err)
	}
	if info.IsDir() {
		return Record{}, nil
	}
	modifiedAt := info.ModTime().UTC()
	record, err := s.repo.UpsertDiscovered(ctx, UpsertParams{
		SourceKind:   SourceKindDirectory,
		OriginalName: filepath.Base(cleanPath),
		SourcePath:   cleanPath,
		CurrentPath:  cleanPath,
		FileSize:     info.Size(),
		ModifiedAt:   &modifiedAt,
		ObservedAt:   s.now(),
	})
	if err != nil {
		return Record{}, err
	}
	s.appendEvent(ctx, traceevent.WriteInput{
		EventType:       "ingest.file.detected",
		EventLevel:      "info",
		Message:         traceevent.Message("Ingest file detected", "Ingest file %s detected", record.OriginalName),
		Actor:           traceevent.Actor{Type: traceevent.ActorWorker, Name: "directory-ingest-worker"},
		SourceComponent: "ingest.service",
		EventData: map[string]any{
			"ingestFileUid": record.UID,
			"path":          record.CurrentPath,
			"status":        record.Status,
		},
	})
	return record, nil
}

func (s *Service) RequeueStaleClaims(ctx context.Context, cfg RuntimeConfig) (int, error) {
	now := s.now()
	return s.repo.RequeueStaleClaims(ctx, RequeueParams{
		StaleBefore: now.Add(-cfg.ClaimTimeout),
		RetryAt:     now,
	})
}

func (s *Service) ProcessBatch(ctx context.Context, exec any, cfg RuntimeConfig) error {
	runID := int64(0)
	if typed, ok := exec.(interface{ RunID() int64 }); ok {
		runID = typed.RunID()
	}
	for index := 0; index < cfg.BatchSize; index++ {
		record, err := s.repo.ClaimNextReady(ctx, ClaimParams{
			WorkerRunID: runID,
			ReadyBefore: s.now().Add(-cfg.Debounce),
			RetryBefore: s.now(),
			ClaimedAt:   s.now(),
		})
		if err != nil {
			if errors.Is(err, ErrNoEligibleFile) {
				return nil
			}
			return err
		}
		if err := s.processOne(ctx, record, cfg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) processOne(ctx context.Context, record Record, cfg RuntimeConfig) error {
	current := record
	processingPath, err := s.moveToProcessing(ctx, current, cfg)
	if err != nil {
		return err
	}
	current.CurrentPath = processingPath

	body, err := s.readFile(processingPath)
	if err != nil {
		return s.failRetryable(ctx, current, cfg, ErrorCodeFileRead, err.Error(), "")
	}
	checksum := checksum(body)

	input, idempotencyKey, err := decodeEnvelope(body, cfg, checksum)
	if err != nil {
		return s.failTerminal(ctx, current, cfg, ErrorCodeInvalidEnvelope, err.Error(), checksum, "")
	}

	created, err := s.requestSvc.CreateRequest(ctx, input)
	if err != nil {
		var appErr *apperror.AppError
		if errors.As(err, &appErr) && appErr.Code == apperror.CodeValidationFailed {
			return s.failTerminal(ctx, current, cfg, appErr.Code, appErr.Message, checksum, idempotencyKey)
		}
		return s.failRetryable(ctx, current, cfg, ErrorCodeRequestCreate, err.Error(), checksum)
	}

	archivedPath, err := s.archiveFile(current.CurrentPath, cfg.ProcessedPath, current.OriginalName, current.UID)
	if err != nil {
		return s.failRetryable(ctx, current, cfg, ErrorCodeFileMove, err.Error(), checksum)
	}
	if _, err := s.repo.MarkProcessed(ctx, MarkProcessedParams{
		ID:             current.ID,
		CurrentPath:    archivedPath,
		ArchivedPath:   archivedPath,
		RequestID:      created.ID,
		ChecksumSHA256: checksum,
		IdempotencyKey: idempotencyKey,
		Meta: map[string]any{
			"requestUid": created.UID,
			"path":       archivedPath,
		},
	}); err != nil {
		return err
	}

	s.logAudit(ctx, audit.Event{
		Action:     "request.ingested_from_directory",
		EntityType: "request",
		EntityID:   strPtr(fmt.Sprintf("%d", created.ID)),
		Metadata: map[string]any{
			"requestUid":     created.UID,
			"ingestFileUid":  current.UID,
			"originalName":   current.OriginalName,
			"idempotencyKey": idempotencyKey,
		},
	})
	s.appendEvent(ctx, traceevent.WriteInput{
		RequestID:       &created.ID,
		EventType:       "ingest.file.processed",
		EventLevel:      "info",
		Message:         traceevent.Message("Ingest file processed", "Ingest file %s created request %s", current.OriginalName, created.UID),
		Actor:           traceevent.Actor{Type: traceevent.ActorWorker, Name: "directory-ingest-worker"},
		SourceComponent: "ingest.service",
		EventData: map[string]any{
			"ingestFileUid":  current.UID,
			"archivedPath":   archivedPath,
			"idempotencyKey": idempotencyKey,
			"checksumSha256": checksum,
		},
	})
	return nil
}

func (s *Service) moveToProcessing(ctx context.Context, record Record, cfg RuntimeConfig) (string, error) {
	if filepath.Clean(filepath.Dir(record.CurrentPath)) == filepath.Clean(cfg.ProcessingPath) {
		return record.CurrentPath, nil
	}
	nextPath, err := uniqueArchivePath(cfg.ProcessingPath, record.OriginalName, record.UID)
	if err != nil {
		return "", s.failRetryable(ctx, record, cfg, ErrorCodeFileMove, err.Error(), "")
	}
	if err := s.ensureDir(filepath.Dir(nextPath)); err != nil {
		return "", s.failRetryable(ctx, record, cfg, ErrorCodeFileMove, err.Error(), "")
	}
	if err := s.rename(record.CurrentPath, nextPath); err != nil {
		return "", s.failRetryable(ctx, record, cfg, ErrorCodeFileMove, err.Error(), "")
	}
	if _, err := s.repo.SetCurrentPath(ctx, SetCurrentPathParams{ID: record.ID, CurrentPath: nextPath}); err != nil {
		return "", err
	}
	s.appendEvent(ctx, traceevent.WriteInput{
		EventType:       "ingest.file.claimed",
		EventLevel:      "info",
		Message:         traceevent.Message("Ingest file claimed", "Ingest file %s claimed for processing", record.OriginalName),
		Actor:           traceevent.Actor{Type: traceevent.ActorWorker, Name: "directory-ingest-worker"},
		SourceComponent: "ingest.service",
		EventData: map[string]any{
			"ingestFileUid": record.UID,
			"path":          nextPath,
		},
	})
	return nextPath, nil
}

func (s *Service) failTerminal(ctx context.Context, record Record, cfg RuntimeConfig, code string, message string, checksumValue string, idempotencyKey string) error {
	archivedPath, moveErr := s.archiveFile(record.CurrentPath, cfg.FailedPath, record.OriginalName, record.UID)
	if moveErr != nil {
		return s.failRetryable(ctx, record, cfg, ErrorCodeFileMove, moveErr.Error(), checksumValue)
	}
	_, err := s.repo.MarkFailed(ctx, MarkFailedParams{
		ID:               record.ID,
		CurrentPath:      archivedPath,
		ArchivedPath:     archivedPath,
		ChecksumSHA256:   checksumValue,
		IdempotencyKey:   idempotencyKey,
		LastErrorCode:    code,
		LastErrorMessage: message,
		Meta: map[string]any{
			"path": archivedPath,
		},
	})
	if err != nil {
		return err
	}
	s.appendEvent(ctx, traceevent.WriteInput{
		EventType:       "ingest.file.failed",
		EventLevel:      "warning",
		Message:         traceevent.Message("Ingest file failed", "Ingest file %s failed: %s", record.OriginalName, message),
		Actor:           traceevent.Actor{Type: traceevent.ActorWorker, Name: "directory-ingest-worker"},
		SourceComponent: "ingest.service",
		EventData: map[string]any{
			"ingestFileUid": record.UID,
			"code":          code,
		},
	})
	return nil
}

func (s *Service) failRetryable(ctx context.Context, record Record, cfg RuntimeConfig, code string, message string, checksumValue string) error {
	_, err := s.repo.MarkRetry(ctx, MarkRetryParams{
		ID:               record.ID,
		CurrentPath:      record.CurrentPath,
		ChecksumSHA256:   checksumValue,
		LastErrorCode:    code,
		LastErrorMessage: message,
		NextAttemptAt:    s.now().Add(cfg.RetryDelay),
		Meta: map[string]any{
			"path": record.CurrentPath,
		},
	})
	if err != nil {
		return err
	}
	s.appendEvent(ctx, traceevent.WriteInput{
		EventType:       "ingest.file.retry_scheduled",
		EventLevel:      "warning",
		Message:         traceevent.Message("Ingest retry scheduled", "Ingest file %s scheduled for retry", record.OriginalName),
		Actor:           traceevent.Actor{Type: traceevent.ActorWorker, Name: "directory-ingest-worker"},
		SourceComponent: "ingest.service",
		EventData: map[string]any{
			"ingestFileUid": record.UID,
			"code":          code,
		},
	})
	return nil
}

type envelope struct {
	SourceSystem         string          `json:"sourceSystem"`
	DestinationServerID  int64           `json:"destinationServerId"`
	DestinationServerIDs []int64         `json:"destinationServerIds"`
	DependencyRequestIDs []int64         `json:"dependencyRequestIds"`
	BatchID              string          `json:"batchId"`
	CorrelationID        string          `json:"correlationId"`
	IdempotencyKey       string          `json:"idempotencyKey"`
	Payload              json.RawMessage `json:"payload"`
	PayloadFormat        string          `json:"payloadFormat"`
	SubmissionBinding    string          `json:"submissionBinding"`
	URLSuffix            string          `json:"urlSuffix"`
	Extras               map[string]any  `json:"extras"`
}

func decodeEnvelope(body []byte, cfg RuntimeConfig, checksumValue string) (requests.CreateInput, string, error) {
	var decoded envelope
	if err := json.Unmarshal(body, &decoded); err != nil {
		return requests.CreateInput{}, "", fmt.Errorf("decode envelope: %w", err)
	}
	idempotencyKey := strings.TrimSpace(decoded.IdempotencyKey)
	if idempotencyKey == "" {
		if cfg.RequireIdempotencyKey {
			return requests.CreateInput{}, "", errors.New("idempotencyKey is required")
		}
		idempotencyKey = "directory:" + checksumValue
	}

	payloadFormat := strings.TrimSpace(decoded.PayloadFormat)
	if payloadFormat == "" {
		payloadFormat = requests.PayloadFormatJSON
	}
	var payload any
	switch payloadFormat {
	case requests.PayloadFormatText:
		var text string
		if err := json.Unmarshal(decoded.Payload, &text); err != nil {
			return requests.CreateInput{}, "", errors.New("payload must be a JSON string when payloadFormat is text")
		}
		payload = text
	default:
		payload = []byte(decoded.Payload)
	}

	extras := cloneAnyMap(decoded.Extras)
	extras["ingestSource"] = SourceKindDirectory
	extras["ingestChecksumSha256"] = checksumValue

	return requests.CreateInput{
		SourceSystem:         firstNonEmpty(strings.TrimSpace(decoded.SourceSystem), strings.TrimSpace(cfg.DefaultSourceSystem)),
		DestinationServerID:  decoded.DestinationServerID,
		DestinationServerIDs: append([]int64{}, decoded.DestinationServerIDs...),
		DependencyRequestIDs: append([]int64{}, decoded.DependencyRequestIDs...),
		BatchID:              strings.TrimSpace(decoded.BatchID),
		CorrelationID:        strings.TrimSpace(decoded.CorrelationID),
		IdempotencyKey:       idempotencyKey,
		Payload:              payload,
		PayloadFormat:        payloadFormat,
		SubmissionBinding:    strings.TrimSpace(decoded.SubmissionBinding),
		URLSuffix:            strings.TrimSpace(decoded.URLSuffix),
		Extras:               extras,
	}, idempotencyKey, nil
}

func checksum(body []byte) string {
	sum := sha256.Sum256(body)
	return hex.EncodeToString(sum[:])
}

func (s *Service) archiveFile(currentPath string, targetDir string, originalName string, uid string) (string, error) {
	targetPath, err := uniqueArchivePath(targetDir, originalName, uid)
	if err != nil {
		return "", err
	}
	if err := s.ensureDir(filepath.Dir(targetPath)); err != nil {
		return "", err
	}
	if err := s.rename(currentPath, targetPath); err != nil {
		return "", err
	}
	return targetPath, nil
}

func uniqueArchivePath(dir string, originalName string, uid string) (string, error) {
	base := filepath.Base(strings.TrimSpace(originalName))
	if base == "." || base == string(os.PathSeparator) || base == "" {
		return "", errors.New("original file name is required")
	}
	ext := filepath.Ext(base)
	name := strings.TrimSuffix(base, ext)
	return filepath.Join(dir, fmt.Sprintf("%s-%s%s", name, strings.ReplaceAll(uid, "-", ""), ext)), nil
}

func extensionAllowed(path string, extensions []string) bool {
	if len(extensions) == 0 {
		return true
	}
	ext := strings.ToLower(filepath.Ext(path))
	for _, allowed := range extensions {
		if strings.ToLower(strings.TrimSpace(allowed)) == ext {
			return true
		}
	}
	return false
}

func cloneAnyMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func (s *Service) ensureDir(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("directory path is required")
	}
	return s.mkdirAll(path, 0o750)
}

func (s *Service) appendEvent(ctx context.Context, input traceevent.WriteInput) {
	if s.eventWriter == nil {
		return
	}
	_ = s.eventWriter.AppendEvent(ctx, input)
}

func (s *Service) logAudit(ctx context.Context, event audit.Event) {
	if s.auditService == nil {
		return
	}
	_ = s.auditService.Log(ctx, event)
}

func strPtr(value string) *string {
	return &value
}
