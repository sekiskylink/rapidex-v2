package dashboard

import (
	"context"
	"time"
)

type Repository interface {
	GetSnapshot(ctx context.Context, now time.Time) (Snapshot, error)
}

type Service struct {
	repository Repository
	now        func() time.Time
	hub        *hub
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
		now:        time.Now,
		hub:        newHub(),
	}
}

func (s *Service) WithClock(now func() time.Time) *Service {
	if now != nil {
		s.now = now
	}
	return s
}

func (s *Service) GetOperationsSnapshot(ctx context.Context) (Snapshot, error) {
	return s.repository.GetSnapshot(ctx, s.now().UTC())
}

func (s *Service) SubscribeOperationsEvents(ctx context.Context) (<-chan StreamEvent, func()) {
	return s.hub.subscribe(ctx)
}

func (s *Service) PublishSourceEvent(_ context.Context, input SourceEvent) {
	s.hub.publish(mapSourceEvent(input))
}
