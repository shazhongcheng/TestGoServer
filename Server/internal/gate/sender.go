// internal/gate/sender.go
package gate

import (
	"game-server/internal/protocol/internalpb"
	"go.uber.org/zap"
)

func (g *Gate) sendToService(module string, env *internalpb.Envelope) {
	if g.servicePool == nil {
		g.logger.Warn("service not initialized")
		return
	}
	if s := g.sessions.Get(env.GetSessionId()); s != nil {
		env.PlayerId = s.PlayerID
	}
	if err := g.servicePool.Send(env.GetSessionId(), env); err != nil {
		g.logger.Warn("send to service failed: %v", zap.Err("err", err))
	}
}

func (g *Gate) sendToGame(env *internalpb.Envelope) {
	g.sendToService("", env)
}
