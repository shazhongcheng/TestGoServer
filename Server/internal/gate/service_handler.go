package gate

import (
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// internal/gate/service_handler.go
func (g *Gate) OnServiceEnvelope(env *internalpb.Envelope) {
	msgID := int(env.MsgId)
	sessionID := env.SessionId

	if env.MsgId == protocol.MsgServicePong {
		// 什么都不用做
		return
	}

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
		fields := append(sessionFields(old),
			zap.Int("msg_id", protocol.MsgLoginRsp),
			zap.String("reason", "duplicate_login"),
		)
		fields = append(fields, connFields(old.Conn)...)
		g.logger.Warn("kick old session", fields...)
		g.Kick(old.ID, "duplicate login")
	}

	s.PlayerID = rsp.PlayerId
	s.State = SessionAuthenticated

	fields := append(sessionFields(s),
		zap.Int("msg_id", protocol.MsgLoginRsp),
		zap.String("reason", "session_authenticated"),
	)
	fields = append(fields, connFields(s.Conn)...)
	g.logger.Info("session authenticated", fields...)

	g.unknownMsgKickCount = 0
}
