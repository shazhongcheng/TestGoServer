package gate

import (
	"game-server/internal/protocol"
	"time"

	"game-server/internal/protocol/internalpb"

	"google.golang.org/protobuf/proto"
)

func (g *Gate) handleResume(c *Conn, env *internalpb.Envelope) {
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

	// 重新绑定
	s.Conn = c
	s.State = SessionOnline
	s.LastSeen = time.Now()
	c.sessionID = s.ID

	g.sendResumeRsp(c, true, "")
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
