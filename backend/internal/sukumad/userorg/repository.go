package userorg

import "context"

// Repository defines persistence behaviour for user‑organisation unit
// assignments.  Implementations should enforce uniqueness of (user_id,
// org_unit_id) and cascade deletes when either the user or the org unit is
// removed.
type Repository interface {
	// ListByUser returns the org unit IDs assigned to a specific user.  If
	// none are found, an empty slice is returned.
	ListByUser(ctx context.Context, userID int64) ([]int64, error)

	// ListAssignmentsByUser returns assignment metadata joined with org units.
	ListAssignmentsByUser(ctx context.Context, userID int64) ([]AssignmentDetail, error)

	// Assign links a user to an org unit.  If the assignment already
	// exists, implementations should be idempotent.  Timestamps should be
	// updated accordingly.
	Assign(ctx context.Context, userID int64, orgUnitID int64) error

	// Remove deletes a specific user→org unit assignment.  If no such
	// assignment exists, implementations should return no error.
	Remove(ctx context.Context, userID int64, orgUnitID int64) error
}
