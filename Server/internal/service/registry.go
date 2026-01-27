// internal/service/registry.go
package service

import (
	"fmt"
	"game-server/internal/handler"
	"sync"
)

type Registry struct {
	modules  map[string]Module
	handlers *handler.Registry[HandlerFunc]
	mu       sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		modules:  make(map[string]Module),
		handlers: handler.NewRegistry[HandlerFunc](),
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

	if err := m.RegisterHandlers(r.handlers); err != nil {
		return fmt.Errorf("register handlers for module %s failed: %w", name, err)
	}

	r.modules[name] = m
	return nil
}

func (r *Registry) GetHandler(msgID int) (HandlerFunc, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	h, ok := r.handlers.Get(msgID)
	return h, ok
}
