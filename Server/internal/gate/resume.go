package gate

import (
	"time"

	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func (g *Gate) handleResume(c *Conn, env *internalpb.Envelope) {
	// 新连接在 Resume 前，必须是未绑定状态
	if c.sessionID != 0 {
		g.logger.Warn("resume on bound conn",
			zap.String("reason", "conn_bound"),
			zap.Int("msg_id", protocol.MsgResumeReq),
			zap.Int64("session", c.sessionID),
			zap.Int64("player", 0),
			zap.Int64("conn_id", c.id),
			zap.String("trace_id", c.traceID),
		)
		c.Close()
		return
	}

	var req internalpb.ResumeReq
	if err := proto.Unmarshal(env.Payload, &req); err != nil {
		c.Close()
		return
	}

	s := g.sessions.Get(req.SessionId)
	if s == nil || !g.verifyToken(s, req.Token) {
		g.sendResumeRsp(c, false, "invalid session")
		fields := append(sessionFields(s),
			zap.Int("msg_id", protocol.MsgResumeReq),
			zap.String("reason", "invalid_session"),
		)
		fields = append(fields, connFields(c)...)
		g.logger.Warn("resume rejected", fields...)
		c.Close()
		return
	}

	// ===== 1️⃣ 状态校验 =====
	switch s.State {
	case SessionAuthing:
		g.sendResumeRsp(c, false, "session authing")
		fields := append(sessionFields(s),
			zap.Int("msg_id", protocol.MsgResumeReq),
			zap.String("reason", "session_authing"),
		)
		fields = append(fields, connFields(c)...)
		g.logger.Warn("resume rejected", fields...)
		c.Close()
		return
	case SessionClosed:
		g.sendResumeRsp(c, false, "session closed")
		fields := append(sessionFields(s),
			zap.Int("msg_id", protocol.MsgResumeReq),
			zap.String("reason", "session_closed"),
		)
		fields = append(fields, connFields(c)...)
		g.logger.Warn("resume rejected", fields...)
		c.Close()
		return
	}

	// ===== 2️⃣ 踢掉旧 Conn（如果存在）=====
	if old := s.Conn; old != nil && old != c {
		fields := append(sessionFields(s),
			zap.Int("msg_id", protocol.MsgResumeReq),
			zap.String("reason", "resume_replace_conn"),
		)
		fields = append(fields, connFields(old)...)
		g.logger.Warn("resume replace old conn", fields...)
		old.Close()
	}

	// ===== 3️⃣ 绑定新 Conn =====
	s.Conn = c
	s.LastSeen = time.Now()

	if s.PlayerID != 0 {
		s.State = SessionAuthenticated
	} else {
		s.State = SessionOnline
	}

	// ===== 4️⃣ 最后一步才写入 =====
	c.sessionID = s.ID

	// ===== 5️⃣ 回包 =====
	g.sendResumeRsp(c, true, "")

	// ===== 6️⃣ 通知 Game =====
	if s.PlayerID != 0 {
		g.notifyPlayerResume(s)
	}

	fields := append(sessionFields(s),
		zap.Int("msg_id", protocol.MsgResumeReq),
		zap.String("reason", "resume_success"),
	)
	fields = append(fields, connFields(c)...)
	g.logger.Info("player resume", fields...)

	g.unknownMsgKickCount = 0
}

func (g *Gate) sendResumeRsp(c *Conn, ok bool, reason string) {
	rsp := &internalpb.ResumeRsp{
		Ok:     ok,
		Reason: reason,
	}

	payload, _ := proto.Marshal(rsp)

	env := &internalpb.Envelope{
		MsgId:     protocol.MsgResumeRsp,
		SessionId: c.sessionID,
		Payload:   payload,
	}

	_ = c.Send(env)
}
