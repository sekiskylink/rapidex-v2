package ratelimit

import (
	"context"
	"time"
)

type Policy struct {
	ID             int64     `db:"id" json:"id"`
	UID            string    `db:"uid" json:"uid"`
	Name           string    `db:"name" json:"name"`
	ScopeType      string    `db:"scope_type" json:"scopeType"`
	ScopeRef       string    `db:"scope_ref" json:"scopeRef"`
	RPS            int       `db:"rps" json:"rps"`
	Burst          int       `db:"burst" json:"burst"`
	MaxConcurrency int       `db:"max_concurrency" json:"maxConcurrency"`
	TimeoutMS      int       `db:"timeout_ms" json:"timeoutMs"`
	IsActive       bool      `db:"is_active" json:"isActive"`
	CreatedAt      time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt      time.Time `db:"updated_at" json:"updatedAt"`
}

type CreateParams struct {
	UID            string
	Name           string
	ScopeType      string
	ScopeRef       string
	RPS            int
	Burst          int
	MaxConcurrency int
	TimeoutMS      int
	IsActive       bool
}

type ListQuery struct {
	Page      int
	PageSize  int
	SortField string
	SortOrder string
}

type ListResult struct {
	Items    []Policy
	Total    int
	Page     int
	PageSize int
}

type Repository interface {
	ListPolicies(context.Context, ListQuery) (ListResult, error)
	GetPolicyByID(context.Context, int64) (Policy, error)
	CreatePolicy(context.Context, CreateParams) (Policy, error)
	FindActivePolicy(context.Context, string, string) (Policy, bool, error)
}
