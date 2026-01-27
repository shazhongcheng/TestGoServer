// internal/gate/sender.go
package gate

import "game-server/internal/protocol/internalpb"

func (g *Gate) sendToService(module string, env *internalpb.Envelope) {
	if g.service == nil {
		g.logger.Warn("service not initialized")
		return
	}
	ctx := g.makeServiceContext(env.GetSessionId(), int(env.GetMsgId()), env.GetPayload())
	g.service.Handle(ctx)
}

func (g *Gate) sendToGame(env *internalpb.Envelope) {
	// TODO: game 服务未接入，预留
}
