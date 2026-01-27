package gate

import (
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"go.uber.org/zap"
)

func (g *Gate) notifyPlayerResume(s *Session) {
	if s == nil || s.PlayerID == 0 {
		return
	}

	env := &internalpb.Envelope{
		MsgId:     protocol.MsgPlayerResumeReq,
		SessionId: s.ID,
		PlayerId:  s.PlayerID,
	}

	g.sendToGame(env)

	fields := append(sessionFields(s), zap.Int("msg_id", protocol.MsgPlayerResumeReq))
	fields = append(fields, zap.String("reason", "player_resume"))
	fields = append(fields, connFields(s.Conn)...)
	g.logger.Info("notify player resume to game", fields...)
}

func (g *Gate) notifyPlayerOffline(s *Session) {
	if s == nil || s.PlayerID == 0 {
		return
	}

	env := &internalpb.Envelope{
		MsgId:     protocol.MsgPlayerOfflineNotify,
		SessionId: s.ID,
		PlayerId:  s.PlayerID,
	}

	g.sendToGame(env)

	fields := append(sessionFields(s), zap.Int("msg_id", protocol.MsgPlayerOfflineNotify))
	fields = append(fields, zap.String("reason", "player_offline"))
	fields = append(fields, connFields(s.Conn)...)
	g.logger.Info("notify player offline to game", fields...)
}
