package gate

import (
	"time"

	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"

	"google.golang.org/protobuf/proto"
)

func (g *Gate) handleResume(c *Conn, env *internalpb.Envelope) {
	// 新连接在 Resume 前，必须是未绑定状态
	if c.sessionID != 0 {
		g.logger.Warn("resume on bound conn")
		c.close()
		return
	}

	var req internalpb.ResumeReq
	if err := proto.Unmarshal(env.Payload, &req); err != nil {
		c.close()
		return
	}

	s := g.sessions.Get(req.SessionId)
	if s == nil || !g.verifyToken(s, req.Token) {
		g.sendResumeRsp(c, false, "invalid session")
		c.close()
		return
	}

	// ===== 1️⃣ 状态校验 =====
	switch s.State {
	case SessionAuthing:
		g.sendResumeRsp(c, false, "session authing")
		c.close()
		return
	case SessionClosed:
		g.sendResumeRsp(c, false, "session closed")
		c.close()
		return
	}

	// ===== 2️⃣ 踢掉旧 Conn（如果存在）=====
	if old := s.Conn; old != nil && old != c {
		g.logger.Warn(
			"resume replace old conn session=%d",
			s.ID,
		)
		old.close()
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

	_ = c.writeEnvelope(env)
}
