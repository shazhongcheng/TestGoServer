// internal/service/module.go
package service

type Module interface {
	Name() string
	Init() error
	Handlers() map[int]HandlerFunc
}
