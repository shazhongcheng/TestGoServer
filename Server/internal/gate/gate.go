package gate

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var ErrSessionNotFound = errors.New("session not found")

type Gate struct {
	logger         *zap.Logger
	debugHeartbeat bool // ⭐ 新增

	sessions *SessionManager

	serviceClient *remoteClient

	heartbeatInterval time.Duration
	heartbeatTimeout  time.Duration
	gcInterval        time.Duration

	nextID int64

	loginTimeout time.Duration

	id        string // 比如 "gate1"
	nextTrace uint64
}

func (g *Gate) newTraceID() string {
	seq := atomic.AddUint64(&g.nextTrace, 1)
	return fmt.Sprintf(
		"%s-%d-%d",
		g.id,
		time.Now().UnixMilli(),
		seq,
	)
}

func NewGate(logger *zap.Logger) *Gate {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Gate{
		logger:            logger,
		sessions:          NewSessionManager(),
		heartbeatInterval: 100 * time.Second,
		heartbeatTimeout:  300 * time.Second,
		gcInterval:        10 * time.Minute,
		loginTimeout:      100 * time.Second,
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
	return s.Conn.Send(env)
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

	_ = conn.Send(&internalpb.Envelope{
		MsgId:     protocol.MsgSessionInit,
		SessionId: s.ID,
		Payload:   data,
	})

	g.logger.Info("session init", append(sessionFields(s), connFields(conn)...)...)
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
	g.onSessionOffline(s, reason)
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

func (g *Gate) Logger() *zap.Logger {
	return g.logger
}

func (g *Gate) onSessionOffline(s *Session, reason string) {
	if s == nil || s.State == SessionClosed {
		return
	}
	if s.State == SessionOffline && s.Conn == nil {
		return
	}
	conn := s.Conn
	if conn != nil {
		conn.Close()
	}
	s.Conn = nil
	s.LastSeen = time.Now()
	wasOnline := s.State != SessionOffline
	s.State = SessionOffline
	if wasOnline {
		g.notifyPlayerOffline(s)
	}
	fields := append(sessionFields(s),
		zap.String("reason", reason),
		zap.Int("msg_id", 0),
	)
	fields = append(fields, connFields(conn)...)
	g.logger.Info("session offline", fields...)
}
