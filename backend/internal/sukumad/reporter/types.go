package reporter

import (
    "time"
)

// Reporter represents a RapidPro contact that has permission to submit reports.
// Reporters are linked to a single organisation unit via OrgUnitID and may be
// deactivated without deletion.
type Reporter struct {
    ID           int64     `db:"id" json:"id"`
    UID          string    `db:"uid" json:"uid"`
    ContactUUID  string    `db:"contact_uuid" json:"contactUuid"`
    PhoneNumber  string    `db:"phone_number" json:"phoneNumber"`
    DisplayName  string    `db:"display_name" json:"displayName"`
    OrgUnitID    int64     `db:"org_unit_id" json:"orgUnitId"`
    IsActive     bool      `db:"is_active" json:"isActive"`
    CreatedAt    time.Time `db:"created_at" json:"createdAt"`
    UpdatedAt    time.Time `db:"updated_at" json:"updatedAt"`
}

// ListQuery defines filters for listing reporters.
type ListQuery struct {
    Page      int
    PageSize  int
    Search    string
    OrgUnitID *int64
    OnlyActive bool
}

// ListResult wraps the paginated Reporter list.
type ListResult struct {
    Items    []Reporter `json:"items"`
    Total    int        `json:"totalCount"`
    Page     int        `json:"page"`
    PageSize int        `json:"pageSize"`
}
