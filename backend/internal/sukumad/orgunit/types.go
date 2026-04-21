package orgunit

import (
    "time"
)

// OrgUnit represents an organisational unit imported from DHIS2 or created locally.
// Each OrgUnit has a materialised path stored in Path which encodes its position in the tree.
// The ParentID refers to the direct parent, or nil if this is a root unit.
type OrgUnit struct {
    ID          int64     `db:"id" json:"id"`
    UID         string    `db:"uid" json:"uid"`
    Code        string    `db:"code" json:"code"`
    Name        string    `db:"name" json:"name"`
    Description string    `db:"description" json:"description"`
    ParentID    *int64    `db:"parent_id" json:"parentId,omitempty"`
    Path        string    `db:"path" json:"path"`
    CreatedAt   time.Time `db:"created_at" json:"createdAt"`
    UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// ListQuery defines filters for listing organisation units.
type ListQuery struct {
    Page     int
    PageSize int
    Search   string
    ParentID *int64
}

// ListResult wraps the paginated OrgUnits.
type ListResult struct {
    Items []OrgUnit `json:"items"`
    Total int       `json:"totalCount"`
    Page  int       `json:"page"`
    PageSize int    `json:"pageSize"`
}
