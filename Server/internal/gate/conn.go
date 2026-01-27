// internal/gate/conn.go
package gate

import "time"

type Conn struct {
	id        int64
	sessionID int64
	gate      *Gate
	closed    chan struct{}
}

func (c *Conn) readLoop() {
	for {
		msgID, payload, err := c.readPacket()
		if err != nil {
			c.gate.onConnClose(c)
			return
		}
		c.gate.OnClientMsg(c.sessionID, msgID, payload)
	}
}

func (g *Gate) onConnClose(c *Conn) {
	s := g.sessions.Get(c.sessionID)
	if s == nil {
		return
	}

	s.Conn = nil
	s.State = SessionOffline
	s.LastSeen = time.Now()

	// 通知 Game / Service：玩家离线（可选）
}

func (g *Gate) onResume(c *Conn, req *ResumeReq) error {
	s := g.sessions.Get(req.SessionId)
	if s == nil {
		return ErrInvalidSession
	}

	if !g.verifyToken(s, req.Token) {
		return ErrInvalidToken
	}

	// 重新绑定
	s.Conn = c
	s.State = SessionOnline
	s.LastSeen = time.Now()
	c.sessionID = s.ID

	return nil
}
