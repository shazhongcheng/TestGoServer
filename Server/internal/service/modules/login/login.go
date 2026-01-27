// internal/service/modules/login/login.go
package login

import "game-server/internal/service"

const (
	MsgLoginReq = 1001
)

type Module struct{}

func (m *Module) Name() string {
	return "login"
}

func (m *Module) Init() error {
	return nil
}

func (m *Module) Handlers() map[int]service.HandlerFunc {
	return map[int]service.HandlerFunc{
		MsgLoginReq: m.onLogin,
	}
}
