package dashboard

import (
	"context"
	"sync"
)

const subscriberBuffer = 32

type hub struct {
	mu          sync.RWMutex
	nextID      int
	subscribers map[int]chan StreamEvent
}

func newHub() *hub {
	return &hub{
		subscribers: map[int]chan StreamEvent{},
	}
}

func (h *hub) subscribe(ctx context.Context) (<-chan StreamEvent, func()) {
	ch := make(chan StreamEvent, subscriberBuffer)

	h.mu.Lock()
	id := h.nextID
	h.nextID++
	h.subscribers[id] = ch
	h.mu.Unlock()

	var once sync.Once
	unsubscribe := func() {
		once.Do(func() {
			h.mu.Lock()
			if existing, ok := h.subscribers[id]; ok {
				delete(h.subscribers, id)
				close(existing)
			}
			h.mu.Unlock()
		})
	}

	go func() {
		<-ctx.Done()
		unsubscribe()
	}()

	return ch, unsubscribe
}

func (h *hub) publish(event StreamEvent) {
	h.mu.RLock()
	subscribers := make([]chan StreamEvent, 0, len(h.subscribers))
	for _, ch := range h.subscribers {
		subscribers = append(subscribers, ch)
	}
	h.mu.RUnlock()

	for _, ch := range subscribers {
		select {
		case ch <- event:
		default:
		}
	}
}
