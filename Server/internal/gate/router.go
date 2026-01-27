package gate

import "game-server/internal/service"

func (g *Gate) makeServiceContext(sessionID int64, msgID int, payload []byte) *service.Context {
	ctx := &service.Context{
		SessionID: sessionID,
		MsgID:     msgID,
		Payload:   payload,
		Reply: func(replyMsgID int, data []byte) error {
			return g.Reply(sessionID, replyMsgID, data)
		},
		Push: func(pushMsgID int, data []byte) error {
			return g.Push(sessionID, pushMsgID, data)
		},
		SetPlayerID: func(playerID int64) {
			g.attachPlayer(sessionID, playerID)
		},
	}
	return ctx
}
