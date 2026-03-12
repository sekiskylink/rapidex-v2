package ratelimit

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Snapshot() StubConfig {
	return StubConfig{Message: "not implemented"}
}
