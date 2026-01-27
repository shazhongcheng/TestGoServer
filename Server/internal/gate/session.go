// internal/gate/session.go
package gate

import "time"

type SessionState int

const (
	SessionInit SessionState = iota
	SessionOnline
	SessionOffline // 断线，等待重连
	SessionClosed  // 彻底销毁
)

type Session struct {
	ID       int64
	PlayerID int64
	Token    string

	State SessionState
	Conn  *Conn

	LastSeen time.Time
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
