package gate

import (
	"context"
	"game-server/internal/protocol"
	"time"
)

func (g *Gate) onHeartbeat(sessionID int64) {
	s := g.sessions.Get(sessionID)
	if s != nil {
		s.LastSeen = time.Now()
	}

	_ = g.Reply(sessionID, protocol.MsgHeartbeatRsp, nil)

	if g.debugHeartbeat {
		g.logger.Debug("heartbeat session=%d", sessionID)
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
			g.logger.Warn("heartbeat timeout", s.ID)
			g.Kick(s.ID, "heartbeat timeout")
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
