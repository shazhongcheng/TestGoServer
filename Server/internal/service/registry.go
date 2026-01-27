// internal/service/registry.go
package service

import (
	"fmt"
	"sync"
)

type Registry struct {
	modules  map[string]Module
	handlers map[int]HandlerFunc
	mu       sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		modules:  make(map[string]Module),
		handlers: make(map[int]HandlerFunc),
	}
}

func (r *Registry) Register(m Module) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := m.Name()
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("module %s already registered", name)
	}

	if err := m.Init(); err != nil {
		return fmt.Errorf("init module %s failed: %w", name, err)
	}

	for msgID, handler := range m.Handlers() {
		if _, exists := r.handlers[msgID]; exists {
			return fmt.Errorf("msgID %d already registered", msgID)
		}
		r.handlers[msgID] = handler
	}

	r.modules[name] = m
	return nil
}

func (r *Registry) GetHandler(msgID int) (HandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	h, ok := r.handlers[msgID]
	return h, ok
}
