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
	if s == nil {
		return
	}

	now := time.Now()
	s.LastSeen = now

	// ⭐ 同时刷新 Conn 的活跃时间
	if s.Conn != nil {
		s.Conn.markAlive(now)
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
		if s.State != SessionOnline || s.Conn == nil {
			continue
		}

		// ⭐ 从 Conn 读取真实活跃时间
		last := s.Conn.lastAlive()
		timeout := g.heartbeatTimeout

		// ⭐ WS 给更宽容的窗口
		if s.Conn.connType == ConnWS {
			timeout += timeout / 2
		}

		if now.Sub(last) <= timeout {
			continue
		}

		fields := append(
			sessionFields(s),
			zap.Int("msg_id", protocol.MsgHeartbeatReq),
			zap.String("reason", "heartbeat_timeout"),
			zap.Duration("idle", now.Sub(last)),
		)
		fields = append(fields, connFields(s.Conn)...)

		g.logger.Warn("heartbeat timeout", fields...)
		atomic.AddUint64(&g.heartbeatTimeoutCount, 1)

		g.onSessionOffline(s, "heartbeat timeout")
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
