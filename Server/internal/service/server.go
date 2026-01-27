// internal/service/server.go
package service

import "go.uber.org/zap"

type Server struct {
	registry   *Registry
	dispatcher *Dispatcher
	logger     *zap.Logger
}

func NewServer(logger *zap.Logger) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}
	reg := NewRegistry()
	return &Server{
		registry:   reg,
		dispatcher: NewDispatcher(reg, logger),
		logger:     logger,
	}
}

func (s *Server) RegisterModule(m Module) error {
	return s.registry.Register(m)
}

func (s *Server) Handle(ctx *Context) {
	s.dispatcher.Dispatch(ctx)
}
