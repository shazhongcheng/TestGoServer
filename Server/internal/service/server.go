// internal/service/server.go
package service

type Server struct {
	registry   *Registry
	dispatcher *Dispatcher
}

func NewServer() *Server {
	reg := NewRegistry()
	return &Server{
		registry:   reg,
		dispatcher: NewDispatcher(reg),
	}
}

func (s *Server) RegisterModule(m Module) error {
	return s.registry.Register(m)
}

func (s *Server) Handle(ctx *Context) {
	s.dispatcher.Dispatch(ctx)
}
