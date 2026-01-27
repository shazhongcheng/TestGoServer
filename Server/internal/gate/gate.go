package gate

import (
	"context"
	"errors"
	"game-server/internal/common/selflog"
	"game-server/internal/protocol/internalpb"
	"sync/atomic"
	"time"

	"game-server/internal/service"
)

var ErrSessionNotFound = errors.New("session not found")

type Gate struct {
	logger         selflog.Logger
	debugHeartbeat bool // ⭐ 新增

	sessions *SessionManager

	service *service.Server

	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	gcInterval        time.Duration

	nextID int64
}

func NewGate(logger selflog.Logger, svc *service.Server) *Gate {
	if logger == nil {
		logger = selflog.NewNopLogger()
	}
	return &Gate{
		logger:            logger,
		sessions:          NewSessionManager(),
		service:           svc,
		heartbeatInterval: 10 * time.Second,
		heartbeatTimeout:  30 * time.Second,
		gcInterval:        1 * time.Minute,
	}
}

func (g *Gate) Start(ctx context.Context) {
	go g.heartbeatLoop(ctx)
	go g.gcLoop(ctx)
}

func (g *Gate) nextSessionID() int64 {
	return atomic.AddInt64(&g.nextID, 1)
}

func (g *Gate) Reply(sessionID int64, msgID int, data []byte) error {
	s := g.sessions.Get(sessionID)
	if s == nil || s.Conn == nil {
		return ErrSessionNotFound
	}

	env := &internalpb.Envelope{
		MsgId:     int32(msgID),
		SessionId: sessionID,
		PlayerId:  s.PlayerID,
		Payload:   data,
	}
	return s.Conn.writeEnvelope(env)
}

func (g *Gate) NewSession(conn *Conn) *Session {
	s := g.newSession()
	s.Conn = conn
	s.State = SessionOnline
	s.LastSeen = time.Now()
	conn.sessionID = s.ID
	g.sessions.Add(s)
	return s
}

func (g *Gate) Push(sessionID int64, msgID int, data []byte) error {
	return g.Reply(sessionID, msgID, data)
}

func (g *Gate) Kick(sessionID int64, reason string) error {
	s := g.sessions.Get(sessionID)
	if s == nil || s.Conn == nil {
		return ErrSessionNotFound
	}

	s.Conn.close()
	return nil
}
