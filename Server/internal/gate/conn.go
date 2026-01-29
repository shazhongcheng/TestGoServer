// internal/gate/conn.go
package gate

import (
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/transport"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

type Conn struct {
	gate *Gate

	conn transport.Conn

	sessionID int64
	traceID   string

	sendCh chan *internalpb.Envelope
	closed chan struct{}
	once   sync.Once

	id int64

	connectedAt time.Time
	busyCount   uint32
}

func NewConn(nc net.Conn, g *Gate) *Conn {
	return NewConnWithTransport(transport.NewBufferedConnWithOptions(nc, g.connOptions), g)
}

func NewWSConn(ws *websocket.Conn, g *Gate, useJSON bool) *Conn {
	return NewConnWithTransport(transport.NewWSConn(ws, useJSON), g)
}

func NewConnWithTransport(conn transport.Conn, g *Gate) *Conn {
	c := &Conn{
		gate: g,
		conn: conn,

		sendCh: make(chan *internalpb.Envelope, 8*1024),
		closed: make(chan struct{}),

		traceID:     g.newTraceID(),
		connectedAt: time.Now(),
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
			if err := c.conn.WriteEnvelope(env); err != nil {
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
		atomic.AddUint64(&c.gate.connBusyCount, 1)
		if atomic.AddUint32(&c.busyCount, 1) >= 5 {
			c.gate.logger.Warn("conn send buffer full, closing",
				zap.String("reason", "conn_busy"),
				zap.String("trace_id", c.traceID),
				zap.Int64("session", c.sessionID),
			)
			c.Close()
		}
		return ErrConnBusy
	}
}

var ErrConnBusy = errors.New("connection send buffer full")

// =======================
// Read Path（阻塞读）
// =======================
func (c *Conn) ReadLoop() {
	for {
		env, err := c.conn.ReadEnvelope()
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
		_ = c.conn.Close()
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
	g.logger.Info("client connection closed",
		zap.String("reason", "read_error"),
		zap.Int64("session", s.ID),
		zap.Int64("player", s.PlayerID),
		zap.Duration("online_duration", time.Since(c.connectedAt)),
		zap.String("trace_id", c.traceID),
	)
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
