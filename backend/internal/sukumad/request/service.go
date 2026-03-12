package request

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) ListResponse() StubResponse {
	return StubResponse{Message: "not implemented"}
}
