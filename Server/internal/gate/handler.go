// internal/gate/handler.go
package gate

import (
	"time"

	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/router"
	"go.uber.org/zap"
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
			g.logger.Warn("reject msg before session init",
				zap.Int("msg_id", msgID),
				zap.String("reason", "session_not_initialized"),
				zap.Int64("session", 0),
				zap.Int64("player", 0),
				zap.Int64("sesson_id", c.sessionID),
				zap.String("trace_id", c.traceID),
			)
			c.Close()
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
		g.logger.Warn("reject msg missing session",
			zap.Int("msg_id", msgID),
			zap.String("reason", "session_not_found"),
			zap.Int64("session", c.sessionID),
			zap.Int64("player", 0),
			zap.Int64("sesson_id", c.sessionID),
			zap.String("trace_id", c.traceID),
		)
		c.Close()
		return
	}
	if s.State == SessionOffline {
		g.logger.Warn("reject msg on offline session",
			zap.Int("msg_id", msgID),
			zap.String("reason", "session_offline"),
			zap.Int64("session", s.ID),
			zap.Int64("player", s.PlayerID),
			zap.Int64("sesson_id", c.sessionID),
			zap.String("trace_id", c.traceID),
		)
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
			g.logger.Warn("duplicate login req",
				zap.String("reason", "duplicate_login"),
				zap.Int("msg_id", msgID),
				zap.Int64("session", s.ID),
				zap.Int64("player", s.PlayerID),
				zap.Int64("sesson_id", c.sessionID),
				zap.String("trace_id", c.traceID),
			)
			return
		}
	}

	// =========================
	// 3️⃣ 权限判断
	// =========================
	if s.State == SessionOnline {
		g.logger.Warn("unauth msg",
			zap.String("reason", "unauthenticated"),
			zap.Int("msg_id", msgID),
			zap.Int64("session", s.ID),
			zap.Int64("player", s.PlayerID),
			zap.Int64("sesson_id", c.sessionID),
			zap.String("trace_id", c.traceID),
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
		g.logger.Warn("unknown msgID",
			zap.Int("msg_id", msgID),
			zap.Int64("session", s.ID),
			zap.Int64("player", s.PlayerID),
			zap.String("reason", "unknown_route"),
			zap.Int64("sesson_id", c.sessionID),
			zap.String("trace_id", c.traceID),
		)
		return
	}

	s.LastSeen = time.Now()

	switch rule.Target {
	case router.TargetService:
		g.sendToService(rule.Module, env)
	case router.TargetGame:
		g.sendToGame(env)
	default:
		g.logger.Warn("unknown route target",
			zap.Int("msg_id", msgID),
			zap.Int64("session", s.ID),
			zap.Int64("player", s.PlayerID),
			zap.String("reason", "unknown_route_target"),
			zap.Int64("sesson_id", c.sessionID),
			zap.String("trace_id", c.traceID),
		)
	}
}

func (g *Gate) handleDuplicateLogin(s *Session) {
	fields := append(sessionFields(s), zap.String("reason", "duplicate_login"), zap.Int("msg_id", protocol.MsgLoginReq))
	fields = append(fields, connFields(s.Conn)...)
	g.logger.Warn("duplicate login kick old session", fields...)
	g.Kick(s.ID, "duplicate login")
}
