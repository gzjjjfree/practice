package core

import (
	"sync"
)

// EventBus 提供轻量级的事件发布/订阅机制，减少状态同步的复杂性。
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[string][]func(data interface{})
}

func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[string][]func(data interface{})),
	}
}

func (eb *EventBus) Subscribe(event string, fn func(data interface{})) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.subscribers[event] = append(eb.subscribers[event], fn)
}

func (eb *EventBus) Publish(event string, data interface{}) {
	eb.mu.RLock()
	fns, exists := eb.subscribers[event]
	eb.mu.RUnlock()
	if !exists {
		return
	}
	for _, fn := range fns {
		go fn(data)
	}
}
