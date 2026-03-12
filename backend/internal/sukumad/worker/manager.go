package worker

type Bootstrap struct {
	Manager *Manager
}

func NewBootstrap(service *Service, defs ...Definition) Bootstrap {
	return Bootstrap{
		Manager: NewManager(service, defs...),
	}
}
