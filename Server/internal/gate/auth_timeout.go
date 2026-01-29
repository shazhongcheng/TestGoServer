// internal/gate/auth_timeout.go
package gate

import (
	"sync/atomic"
	"time"

	"game-server/internal/protocol"
	"go.uber.org/zap"
)

func (g *Gate) checkAuthingTimeout() {
	now := time.Now()

	for _, s := range g.sessions.snapshot() {
		if s.State != SessionAuthing {
			continue
		}

		if now.Sub(s.AuthStart) > g.loginTimeout {
			fields := append(sessionFields(s),
				zap.Int("msg_id", protocol.MsgLoginReq),
				zap.String("reason", "login_timeout"),
			)
			fields = append(fields, connFields(s.Conn)...)
			g.logger.Warn("login timeout", fields...)
			atomic.AddUint64(&g.loginTimeoutCount, 1)
			g.onSessionOffline(s, "login timeout")
		}
	}
}
