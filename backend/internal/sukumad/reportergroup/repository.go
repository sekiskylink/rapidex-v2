package reportergroup

import "context"

type Repository interface {
	List(context.Context, ListQuery) (ListResult, error)
	GetByID(context.Context, int64) (ReporterGroup, error)
	GetByName(context.Context, string) (ReporterGroup, error)
	ListByNames(context.Context, []string, bool) ([]ReporterGroup, error)
	Create(context.Context, ReporterGroup) (ReporterGroup, error)
	Update(context.Context, ReporterGroup) (ReporterGroup, error)
}
