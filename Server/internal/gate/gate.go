package gate

import (
	"context"
	"errors"
	"game-server/internal/protocol"
	"google.golang.org/protobuf/proto"
	"sync/atomic"
	"time"

	"game-server/internal/common/selflog"
	"game-server/internal/protocol/internalpb"
)

var ErrSessionNotFound = errors.New("session not found")

type Gate struct {
	logger         selflog.Logger
	debugHeartbeat bool // ⭐ 新增

	sessions *SessionManager

	serviceClient *remoteClient

	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	gcInterval        time.Duration

	nextID int64

	loginTimeout time.Duration
}

func NewGate(logger selflog.Logger) *Gate {
	if logger == nil {
		logger = selflog.NewNopLogger()
	}
	return &Gate{
		logger:            logger,
		sessions:          NewSessionManager(),
		heartbeatInterval: 10 * time.Second,
		heartbeatTimeout:  30 * time.Second,
		gcInterval:        1 * time.Minute,
		loginTimeout:      10 * time.Second,
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

	// ⭐ 核心：SessionInit
	init := &internalpb.SessionInit{
		SessionId: s.ID,
		Token:     s.Token, // 现在可以 mock
	}
	data, _ := proto.Marshal(init)

	_ = conn.writeEnvelope(&internalpb.Envelope{
		MsgId:     protocol.MsgSessionInit,
		SessionId: s.ID,
		Payload:   data,
	})

	g.logger.Info("session init session=%d", s.ID)
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

func (g *Gate) ConnectService(ctx context.Context, addr string) {
	g.serviceClient = newRemoteClient("service", addr, g.logger, g.OnServiceEnvelope)
	g.serviceClient.Start(ctx)
}

func (g *Gate) UpdateConfig(interval, timeout, gc time.Duration) {
	if interval > 0 {
		g.heartbeatInterval = interval
	}
	if timeout > 0 {
		g.heartbeatTimeout = timeout
	}
	if gc > 0 {
		g.gcInterval = gc
	}
}
