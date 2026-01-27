package gate

import (
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
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

	g.logger.Info(
		"notify player resume to game session=%d player=%d",
		s.ID, s.PlayerID,
	)
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

	g.logger.Info(
		"notify player offline to game session=%d player=%d",
		s.ID, s.PlayerID,
	)
}
