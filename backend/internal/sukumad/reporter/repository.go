package reporter

import (
	"context"
	"time"
)

// Repository defines persistence behaviour for reporters.  Implementations should
// enforce required reporter persistence and lookup behaviour.
type Repository interface {
	// List returns a page of Reporters matching the provided query.  The search string
	// should be matched against phone numbers and display names.  If OrgUnitID is
	// provided, only reporters attached to that unit should be returned.
	List(ctx context.Context, query ListQuery) (ListResult, error)

	// ListBroadcasts returns a page of queued/running/completed/failed jurisdiction broadcasts.
	ListBroadcasts(ctx context.Context, query BroadcastListQuery) (BroadcastListResult, error)

	// GetByID fetches a reporter by its numeric ID.
	GetByID(ctx context.Context, id int64) (Reporter, error)

	// GetByUID fetches a reporter by its UID.
	GetByUID(ctx context.Context, uid string) (Reporter, error)

	// GetByRapidProUUID fetches a reporter by its RapidPro contact UUID.
	GetByRapidProUUID(ctx context.Context, rapidProUUID string) (Reporter, error)

	// GetByPhoneNumber fetches a reporter by its phone number.
	GetByPhoneNumber(ctx context.Context, phone string) (Reporter, error)

	// ListByIDs fetches reporters by IDs.
	ListByIDs(ctx context.Context, ids []int64) ([]Reporter, error)

	// ListUpdatedSince fetches reporters created or updated after the provided time.
	ListUpdatedSince(ctx context.Context, since *time.Time, limit int, onlyActive bool) ([]Reporter, error)

	// CountBroadcastRecipients returns the number of reporters matching the broadcast query.
	CountBroadcastRecipients(ctx context.Context, query BroadcastRecipientQuery) (int, error)

	// ListBroadcastRecipients returns reporters matching the broadcast query.
	ListBroadcastRecipients(ctx context.Context, query BroadcastRecipientQuery) ([]Reporter, error)

	// GetRecentPendingBroadcastByDedupeKey returns a recent queued or running broadcast with the same dedupe key.
	GetRecentPendingBroadcastByDedupeKey(ctx context.Context, dedupeKey string, since time.Time) (JurisdictionBroadcastRecord, error)

	// CreateJurisdictionBroadcast enqueues a new jurisdiction broadcast request.
	CreateJurisdictionBroadcast(ctx context.Context, record JurisdictionBroadcastRecord) (JurisdictionBroadcastRecord, error)

	// ClaimNextJurisdictionBroadcast claims the next queued broadcast for worker processing.
	ClaimNextJurisdictionBroadcast(ctx context.Context, now time.Time, claimTimeout time.Duration, workerRunID int64) (JurisdictionBroadcastRecord, error)

	// UpdateJurisdictionBroadcastResult persists terminal processing state for a broadcast.
	UpdateJurisdictionBroadcastResult(ctx context.Context, id int64, status string, sentCount int, failedCount int, lastError string, finishedAt time.Time) (JurisdictionBroadcastRecord, error)

	// UpdateRapidProStatus updates RapidPro linkage fields without mutating the local change timestamp watermark.
	UpdateRapidProStatus(ctx context.Context, id int64, rapidProUUID string, synced bool) (Reporter, error)

	// Create inserts a new reporter and returns the persisted record.
	Create(ctx context.Context, reporter Reporter) (Reporter, error)

	// Update modifies an existing reporter.  The ID must be set.  Only display name,
	// org unit, phone number and active flag may be changed.
	Update(ctx context.Context, reporter Reporter) (Reporter, error)

	// Delete permanently removes a reporter.
	Delete(ctx context.Context, id int64) error
}
