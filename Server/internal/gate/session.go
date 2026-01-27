// internal/gate/session.go
package gate

import (
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"google.golang.org/protobuf/proto"
	"time"
)

type SessionState int

const (
	SessionInit SessionState = iota
	SessionOnline
	SessionAuthing
	SessionAuthenticated // ✅ 已登录
	SessionOffline       // 断线，等待重连
	SessionClosed        // 彻底销毁
)

type Session struct {
	ID       int64
	PlayerID int64
	Token    string

	State SessionState
	Conn  *Conn

	LastSeen time.Time

	// ⭐ 登录相关
	AuthStart time.Time
}

func (g *Gate) newSession() *Session {
	s := &Session{
		ID:       g.nextSessionID(),
		State:    SessionInit,
		LastSeen: time.Now(),
	}
	s.Token = g.signResumeToken(s)
	return s
}

func (s *Session) MarkSeen() {
	s.LastSeen = time.Now()
}

func (g *Gate) createSessionForConn(c *Conn) *Session {
	s := g.newSession()
	s.Conn = c
	s.State = SessionOnline
	s.LastSeen = time.Now()

	c.sessionID = s.ID
	g.sessions.Add(s)

	init := &internalpb.SessionInit{
		SessionId: s.ID,
		Token:     s.Token,
	}
	data, _ := proto.Marshal(init)

	_ = c.writeEnvelope(&internalpb.Envelope{
		MsgId:     protocol.MsgSessionInit,
		SessionId: s.ID,
		Payload:   data,
	})

	g.logger.Info("session init session=%d", s.ID)
	return s
}
