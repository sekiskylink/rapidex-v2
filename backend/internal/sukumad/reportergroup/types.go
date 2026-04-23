package reportergroup

import "time"

type ReporterGroup struct {
	ID        int64     `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	IsActive  bool      `db:"is_active" json:"isActive"`
	CreatedAt time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt time.Time `db:"updated_at" json:"updatedAt"`
}

type ListQuery struct {
	Page       int
	PageSize   int
	Search     string
	ActiveOnly bool
}

type ListResult struct {
	Items    []ReporterGroup `json:"items"`
	Total    int             `json:"totalCount"`
	Page     int             `json:"page"`
	PageSize int             `json:"pageSize"`
}

type Option struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}
