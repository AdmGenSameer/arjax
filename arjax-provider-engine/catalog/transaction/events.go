package transaction

import "sync"

// UpdateHandler receives transaction lifecycle snapshots cleanly
type UpdateHandler func(TransactionDTO)

// EventBus fans out transaction updates to registered listeners
type EventBus struct {
	mu        sync.RWMutex
	listeners []UpdateHandler
}

func NewEventBus() *EventBus {
	return &EventBus{}
}

func (eb *EventBus) Subscribe(fn UpdateHandler) {
	if fn == nil {
		return
	}
	eb.mu.Lock()
	eb.listeners = append(eb.listeners, fn)
	eb.mu.Unlock()
}

func (eb *EventBus) Emit(dto TransactionDTO) {
	eb.mu.RLock()
	list := append([]UpdateHandler(nil), eb.listeners...)
	eb.mu.RUnlock()
	for _, fn := range list {
		fn(dto)
	}
}

