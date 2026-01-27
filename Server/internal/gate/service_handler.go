package gate

import (
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"google.golang.org/protobuf/proto"
)

// internal/gate/service_handler.go
func (g *Gate) OnServiceEnvelope(env *internalpb.Envelope) {
	msgID := int(env.MsgId)
	sessionID := env.SessionId

	switch msgID {
	case protocol.MsgLoginRsp:
		g.onLoginRsp(sessionID, env.Payload)
	}

	// 默认：原样转发给客户端
	_ = g.Reply(sessionID, msgID, env.Payload)
}

func (g *Gate) onLoginRsp(sessionID int64, payload []byte) {
	s := g.sessions.Get(sessionID)
	if s == nil {
		return
	}

	var rsp internalpb.LoginRsp
	if err := proto.Unmarshal(payload, &rsp); err != nil {
		return
	}

	// ⭐ 顶号处理
	old := g.sessions.BindPlayer(s, rsp.PlayerId)
	if old != nil && old.ID != s.ID {
		g.logger.Warn(
			"kick old session=%d player=%d",
			old.ID, rsp.PlayerId,
		)
		g.Kick(old.ID, "duplicate login")
	}

	s.PlayerID = rsp.PlayerId
	s.State = SessionAuthenticated

	g.logger.Info(
		"session authenticated session=%d player=%d",
		s.ID, s.PlayerID,
	)
}
