package server

import (
	"context"
	"time"
)

type Record struct {
	ID                      int64             `db:"id" json:"id"`
	UID                     string            `db:"uid" json:"uid"`
	Name                    string            `db:"name" json:"name"`
	Code                    string            `db:"code" json:"code"`
	SystemType              string            `db:"system_type" json:"systemType"`
	BaseURL                 string            `db:"base_url" json:"baseUrl"`
	EndpointType            string            `db:"endpoint_type" json:"endpointType"`
	HTTPMethod              string            `db:"http_method" json:"httpMethod"`
	UseAsync                bool              `db:"use_async" json:"useAsync"`
	ParseResponses          bool              `db:"parse_responses" json:"parseResponses"`
	ResponseBodyPersistence string            `db:"response_body_persistence" json:"responseBodyPersistence"`
	Headers                 map[string]string `json:"headers"`
	URLParams               map[string]string `json:"urlParams"`
	Suspended               bool              `db:"suspended" json:"suspended"`
	CreatedAt               time.Time         `db:"created_at" json:"createdAt"`
	UpdatedAt               time.Time         `db:"updated_at" json:"updatedAt"`
	CreatedBy               *int64            `db:"created_by" json:"createdBy,omitempty"`
}

type ListQuery struct {
	Page      int
	PageSize  int
	SortField string
	SortOrder string
	Filter    string
}

type ListResult struct {
	Items    []Record
	Total    int
	Page     int
	PageSize int
}

type CreateParams struct {
	UID                     string
	Name                    string
	Code                    string
	SystemType              string
	BaseURL                 string
	EndpointType            string
	HTTPMethod              string
	UseAsync                bool
	ParseResponses          bool
	ResponseBodyPersistence string
	Headers                 map[string]string
	URLParams               map[string]string
	Suspended               bool
	CreatedBy               *int64
}

type UpdateParams struct {
	ID                      int64
	Name                    string
	Code                    string
	SystemType              string
	BaseURL                 string
	EndpointType            string
	HTTPMethod              string
	UseAsync                bool
	ParseResponses          bool
	ResponseBodyPersistence string
	Headers                 map[string]string
	URLParams               map[string]string
	Suspended               bool
}

type Repository interface {
	ListServers(ctx context.Context, query ListQuery) (ListResult, error)
	GetServerByID(ctx context.Context, id int64) (Record, error)
	GetServerByUID(ctx context.Context, uid string) (Record, error)
	CreateServer(ctx context.Context, params CreateParams) (Record, error)
	UpdateServer(ctx context.Context, params UpdateParams) (Record, error)
	DeleteServer(ctx context.Context, id int64) error
}

type CreateInput struct {
	Name                    string
	Code                    string
	SystemType              string
	BaseURL                 string
	EndpointType            string
	HTTPMethod              string
	UseAsync                bool
	ParseResponses          bool
	ResponseBodyPersistence string
	Headers                 map[string]string
	URLParams               map[string]string
	Suspended               bool
	ActorID                 *int64
}

type UpdateInput struct {
	ID                      int64
	Name                    string
	Code                    string
	SystemType              string
	BaseURL                 string
	EndpointType            string
	HTTPMethod              string
	UseAsync                bool
	ParseResponses          bool
	ResponseBodyPersistence string
	Headers                 map[string]string
	URLParams               map[string]string
	Suspended               bool
	ActorID                 *int64
}
