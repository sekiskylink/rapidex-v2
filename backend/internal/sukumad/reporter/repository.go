package reporter

import "context"

// Repository defines persistence behaviour for reporters.  Implementations should
// enforce uniqueness of contact UUIDs and phone numbers.
type Repository interface {
    // List returns a page of Reporters matching the provided query.  The search string
    // should be matched against phone numbers and display names.  If OrgUnitID is
    // provided, only reporters attached to that unit should be returned.
    List(ctx context.Context, query ListQuery) (ListResult, error)

    // GetByID fetches a reporter by its numeric ID.
    GetByID(ctx context.Context, id int64) (Reporter, error)

    // GetByUID fetches a reporter by its UID.
    GetByUID(ctx context.Context, uid string) (Reporter, error)

    // GetByContactUUID fetches a reporter by its RapidPro contact UUID.
    GetByContactUUID(ctx context.Context, contactUUID string) (Reporter, error)

    // GetByPhoneNumber fetches a reporter by its phone number.
    GetByPhoneNumber(ctx context.Context, phone string) (Reporter, error)

    // Create inserts a new reporter and returns the persisted record.
    Create(ctx context.Context, reporter Reporter) (Reporter, error)

    // Update modifies an existing reporter.  The ID must be set.  Only display name,
    // org unit, phone number and active flag may be changed.
    Update(ctx context.Context, reporter Reporter) (Reporter, error)

    // Delete permanently removes a reporter.
    Delete(ctx context.Context, id int64) error
}
