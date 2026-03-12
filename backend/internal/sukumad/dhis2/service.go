package dhis2

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Snapshot() ClientConfig {
	return ClientConfig{Message: "not implemented"}
}
