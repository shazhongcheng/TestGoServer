// internal/gate/handler.go
package gate

import (
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/router"
	"time"
)

func (g *Gate) OnEnvelope(c *Conn, env *internalpb.Envelope) {
	msgID := int(env.MsgId)

	// =========================
	// 1️⃣ Session 尚未绑定：只允许 Resume
	// =========================
	if c.sessionID == 0 {
		if msgID != protocol.MsgResumeReq {
			g.logger.Warn("reject msg before resume, msgID=%d", msgID)
			c.close()
			return
		}
		g.handleResume(c, env)
		return
	}

	sessionID := c.sessionID
	if s := g.sessions.Get(sessionID); s != nil {
		if s.State != SessionAuthenticated &&
			msgID != protocol.MsgLoginReq {
			g.logger.Warn("unauth msg session=%d msgID=%d", sessionID, msgID)
			return
		}

		//TODO 有问题 需要处理为踢玩家！
		if msgID == protocol.MsgLoginReq && s.State == SessionAuthenticated {
			g.logger.Warn("duplicate login session=%d player=%d", s.ID, s.PlayerID)
			return
		}
	}

	// =========================
	// 2️⃣ Gate 控制消息（不进 Router）
	// =========================
	switch msgID {

	case protocol.MsgHeartbeatReq:
		g.onHeartbeat(sessionID)
		return
	}

	// =========================
	// 3️⃣ 业务消息 → Router
	// =========================
	rule, ok := router.GetRoute(msgID)
	if !ok {
		g.logger.Warn("unknown msgID=%d", msgID)
		return
	}

	// 更新活跃时间（业务消息也算活跃）
	if s := g.sessions.Get(sessionID); s != nil {
		s.LastSeen = time.Now()
	}

	switch rule.Target {
	case router.TargetService:
		g.sendToService(rule.Module, env)

	case router.TargetGame:
		g.sendToGame(env)

	default:
		g.logger.Warn("unknown route target=%v msgID=%d", rule.Target, msgID)
	}
}
