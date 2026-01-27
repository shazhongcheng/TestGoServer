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
	// 1️⃣ Resume 协商：优先处理
	// =========================
	if msgID == protocol.MsgResumeReq {
		g.handleResume(c, env)
		return
	}

	if c.sessionID == 0 {
		if msgID != protocol.MsgLoginReq {
			g.logger.Warn("reject msg before session init msgID=%d", msgID)
			c.close()
			return
		}

		// ⭐ 只有这里才创建 Session
		s := g.createSessionForConn(c)

		// 继续走 Login 流程
		g.sendToService("login", &internalpb.Envelope{
			MsgId:     int32(msgID),
			SessionId: s.ID,
			Payload:   env.Payload,
		})
		return
	}

	s := g.sessions.Get(c.sessionID)
	if s == nil {
		c.close()
		return
	}

	// =========================
	// 2️⃣ Login 流程
	// =========================
	if msgID == protocol.MsgLoginReq {
		switch s.State {

		case SessionAuthenticated:
			// 重复登录策略（下面第四部分讲）
			g.handleDuplicateLogin(s)
			return

		case SessionOnline:
			// ⭐ 进入 Authing
			s.State = SessionAuthing
			s.AuthStart = time.Now()

		case SessionAuthing:
			// 忽略重复 LoginReq
			g.logger.Warn("duplicate login req session=%d", s.ID)
			return
		}
	}

	// =========================
	// 3️⃣ 权限判断
	// =========================
	if s.State == SessionOnline {
		g.logger.Warn(
			"unauth msg session=%d msgID=%d",
			s.ID, msgID,
		)
		return
	}

	// =========================
	// 4️⃣ Gate 控制消息
	// =========================
	switch msgID {
	case protocol.MsgHeartbeatReq:
		g.onHeartbeat(s.ID)
		return
	}

	// =========================
	// 5️⃣ 业务路由
	// =========================
	rule, ok := router.GetRoute(msgID)
	if !ok {
		g.logger.Warn("unknown msgID=%d", msgID)
		return
	}

	s.LastSeen = time.Now()

	switch rule.Target {
	case router.TargetService:
		g.sendToService(rule.Module, env)
	case router.TargetGame:
		g.sendToGame(env)
	default:
		g.logger.Warn("unknown route target=%v msgID=%d", rule.Target, msgID)
	}
}

func (g *Gate) handleDuplicateLogin(s *Session) {
	g.logger.Warn(
		"duplicate login kick old session=%d player=%d",
		s.ID, s.PlayerID,
	)
	g.Kick(s.ID, "duplicate login")
}
