package login

import "game-server/internal/service"

func (m *Module) onLogin(ctx *service.Context) error {
	// TODO: 校验 token
	// TODO: 访问 DB
	ctx.Reply(1002, []byte("login ok"))
	return nil
}
