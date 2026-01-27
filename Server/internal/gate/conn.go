// internal/gate/conn.go
package gate

import (
	"errors"
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/transport"
	"net"
	"sync"
	"time"
)

type Conn struct {
	gate *Gate

	rawConn net.Conn
	bc      *transport.BufferedConn

	sessionID int64
	traceID   string

	sendCh chan *internalpb.Envelope
	closed chan struct{}
	once   sync.Once

	id int64
}

func NewConn(nc net.Conn, g *Gate) *Conn {
	c := &Conn{
		gate:    g,
		rawConn: nc,
		bc:      transport.NewBufferedConn(nc),

		sendCh: make(chan *internalpb.Envelope, 1024),
		closed: make(chan struct{}),

		traceID: g.newTraceID(),
	}

	go c.writeLoop()
	return c
}

func (c *Conn) SessonId() int64 {
	return c.sessionID
}

func (c *Conn) TraceID() string {
	return c.traceID
}

func (c *Conn) writeLoop() {
	defer c.Close()

	for {
		select {
		case env := <-c.sendCh:
			if err := c.bc.WriteEnvelope(env); err != nil {
				return
			}

		case <-c.closed:
			return
		}
	}
}

func (c *Conn) Send(env *internalpb.Envelope) error {
	select {
	case c.sendCh <- env:
		return nil
	default:
		return ErrConnBusy
	}
}

var ErrConnBusy = errors.New("connection send buffer full")

// =======================
// Read Path（阻塞读）
// =======================
func (c *Conn) ReadLoop() {
	for {
		env, err := c.bc.ReadEnvelope()
		if err != nil {
			c.gate.onConnClose(c)
			return
		}

		c.gate.OnEnvelope(c, env)
	}
}

// =======================
// Lifecycle
// =======================
func (c *Conn) Close() {
	c.once.Do(func() {
		close(c.closed)
		_ = c.rawConn.Close()
	})
}

// =======================
// Gate hooks
// =======================
func (g *Gate) onConnClose(c *Conn) {
	s := g.sessions.Get(c.sessionID)
	if s == nil {
		return
	}

	s.Conn = nil
	s.State = SessionOffline
	s.LastSeen = time.Now()

	g.notifyPlayerOffline(s)
}

func (g *Gate) onResume(c *Conn, req *ResumeReq) error {
	s := g.sessions.Get(req.SessionId)
	if s == nil {
		return protocol.InternalErrInvalidSession
	}

	if !g.verifyToken(s, req.Token) {
		return protocol.InternalErrInvalidToken
	}

	// 重新绑定
	s.Conn = c
	s.State = SessionOnline
	s.LastSeen = time.Now()
	c.sessionID = s.ID

	return nil
}

func (g *Gate) attachPlayer(sessionID int64, playerID int64) {
	s := g.sessions.Get(sessionID)
	if s == nil {
		return
	}
	if s.PlayerID == playerID {
		return
	}
	s.PlayerID = playerID
}
