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
