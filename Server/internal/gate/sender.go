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
	traceID := ""
	if s := g.sessions.Get(env.GetSessionId()); s != nil {
		env.PlayerId = s.PlayerID
		if s.Conn != nil {
			traceID = s.Conn.TraceID()
		}
	}
	if err := g.servicePool.Send(env.GetSessionId(), env); err != nil {
		g.logger.Warn("send to service failed",
			zap.String("reason", err.Error()),
			zap.Int("msg_id", int(env.MsgId)),
			zap.Int64("session", env.SessionId),
			zap.Int64("player", env.PlayerId),
			zap.String("trace_id", traceID),
		)
	}
}

func (g *Gate) sendToGame(env *internalpb.Envelope) {
	g.sendToService("", env)
}
