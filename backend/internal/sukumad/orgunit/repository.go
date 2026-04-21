package orgunit

import "context"

// Repository defines persistence behaviour for organisation units.
// Implementations should encapsulate database queries and provide context cancellation support.
type Repository interface {
    // List returns a page of OrgUnits matching the provided query.  The search string
    // should be matched against code and name.  If ParentID is provided, only children of
    // that parent should be returned.  Page numbering is zero based.
    List(ctx context.Context, query ListQuery) (ListResult, error)

    // GetByID fetches an OrgUnit by its numeric ID.
    GetByID(ctx context.Context, id int64) (OrgUnit, error)

    // GetByUID fetches an OrgUnit by its UID.
    GetByUID(ctx context.Context, uid string) (OrgUnit, error)

    // GetByCode fetches an OrgUnit by its code.
    GetByCode(ctx context.Context, code string) (OrgUnit, error)

    // Create inserts a new OrgUnit and returns the persisted record.
    Create(ctx context.Context, unit OrgUnit) (OrgUnit, error)

    // Update modifies an existing OrgUnit.  The ID field must be set.  Only code, name,
    // description and parent may be changed.
    Update(ctx context.Context, unit OrgUnit) (OrgUnit, error)

    // Delete removes an OrgUnit.  Implementations must prevent deletion if any child
    // units exist; they should return a suitable error instead.
    Delete(ctx context.Context, id int64) error
}
