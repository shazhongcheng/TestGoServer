// internal/gate/handler.go
package gate

import (
	"game-server/internal/protocol/internalpb"
	"game-server/internal/router"
)

func (g *Gate) OnClientMsg(sessionID int64, msgID int, payload []byte) {
	rule, ok := router.GetRoute(msgID)
	if !ok {
		g.logger.Warn("unknown msgID", msgID)
		return
	}

	env := &internalpb.Envelope{
		MsgId:     int32(msgID),
		SessionId: sessionID,
		Payload:   payload,
	}

	switch rule.Target {
	case router.TargetService:
		g.sendToService(rule.Module, env)
	case router.TargetGame:
		g.sendToGame(env)
	default:
		g.logger.Warn("unknown route target", rule.Target)
	}
}
