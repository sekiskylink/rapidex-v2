package userorg

import "time"

// UserOrgUnit links a user to an organisation unit.  A user may be assigned
// multiple organisation units.  The application should treat these
// assignments as roots when determining the full scope (including
// descendants).  CreatedAt and UpdatedAt timestamps enable auditing of
// assignment changes.
type UserOrgUnit struct {
	UserID    int64     `db:"user_id" json:"userId"`
	OrgUnitID int64     `db:"org_unit_id" json:"orgUnitId"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

// AssignmentRequest represents the payload for assigning a user to an org unit.
type AssignmentRequest struct {
	UserID    int64 `json:"userId"`
	OrgUnitID int64 `json:"orgUnitId"`
}

// AssignmentDetail carries enough information for admin UIs and scope
// resolution without requiring extra org-unit lookups.
type AssignmentDetail struct {
	OrgUnitID   int64  `db:"org_unit_id" json:"orgUnitId"`
	OrgUnitName string `db:"org_unit_name" json:"orgUnitName"`
	OrgUnitPath string `db:"org_unit_path" json:"orgUnitPath"`
	DisplayPath string `db:"display_path" json:"displayPath"`
}

// Scope represents the effective org-unit jurisdiction for a user.
// When Restricted is false, the caller has unrestricted access.
type Scope struct {
	Restricted         bool    `json:"restricted"`
	AssignedOrgUnitIDs []int64 `json:"assignedOrgUnitIds"`
	PathPrefixes       []string
}
