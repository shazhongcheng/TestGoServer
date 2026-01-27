// internal/service/module.go
package service

import "game-server/internal/handler"

type Module interface {
	Name() string
	Init() error
	RegisterHandlers(reg *handler.Registry[HandlerFunc]) error
}
