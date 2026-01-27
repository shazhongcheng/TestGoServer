package handler

import (
	"fmt"
	"sync"
)

// Registry provides a shared msgID -> handler registry.
type Registry[T any] struct {
	handlers map[int]T
	mu       sync.RWMutex
}

func NewRegistry[T any]() *Registry[T] {
	return &Registry[T]{
		handlers: make(map[int]T),
	}
}

func (r *Registry[T]) Register(msgID int, handler T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.handlers[msgID]; exists {
		return fmt.Errorf("msgID %d already registered", msgID)
	}
	r.handlers[msgID] = handler
	return nil
}

func (r *Registry[T]) Get(msgID int) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	h, ok := r.handlers[msgID]
	return h, ok
}
