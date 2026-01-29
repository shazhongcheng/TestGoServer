package gate

import (
	"context"
	"sync/atomic"
	"time"

	"game-server/internal/protocol"
	"go.uber.org/zap"
)

func (g *Gate) onHeartbeat(sessionID int64) {
	s := g.sessions.Get(sessionID)
	if s != nil {
		s.LastSeen = time.Now()
	}

	_ = g.Reply(sessionID, protocol.MsgHeartbeatRsp, nil)

	if g.debugHeartbeat {
		var conn *Conn
		if s != nil {
			conn = s.Conn
		}
		fields := append(sessionFields(s), zap.Int("msg_id", protocol.MsgHeartbeatReq))
		fields = append(fields, zap.String("reason", "heartbeat"))
		fields = append(fields, connFields(conn)...)
		g.logger.Debug("heartbeat", fields...)
	}
}

func (g *Gate) heartbeatLoop(ctx context.Context) {
	if g.heartbeatInterval <= 0 {
		return
	}

	ticker := time.NewTicker(g.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.checkHeartbeat()
		}
	}
}

func (g *Gate) checkHeartbeat() {
	if g.heartbeatTimeout <= 0 {
		return
	}

	now := time.Now()
	for _, s := range g.sessions.snapshot() {
		if s.State != SessionOnline {
			continue
		}
		if now.Sub(s.LastSeen) > g.heartbeatTimeout {
			fields := append(sessionFields(s),
				zap.Int("msg_id", protocol.MsgHeartbeatReq),
				zap.String("reason", "heartbeat_timeout"),
			)
			fields = append(fields, connFields(s.Conn)...)
			g.logger.Warn("heartbeat timeout", fields...)
			atomic.AddUint64(&g.heartbeatTimeoutCount, 1)
			g.onSessionOffline(s, "heartbeat timeout")
		}
	}
}

func (g *Gate) gcLoop(ctx context.Context) {
	if g.gcInterval <= 0 {
		return
	}

	ticker := time.NewTicker(g.gcInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			g.checkAuthingTimeout()
			g.sessions.GC(g.heartbeatTimeout)
		}
	}
}
