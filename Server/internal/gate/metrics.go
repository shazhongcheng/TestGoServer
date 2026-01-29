package gate

import (
	"context"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

func (g *Gate) reportStats(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = time.Minute
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			heartbeatTimeouts := atomic.SwapUint64(&g.heartbeatTimeoutCount, 0)
			loginTimeouts := atomic.SwapUint64(&g.loginTimeoutCount, 0)
			loginLimited := atomic.SwapUint64(&g.loginRateLimitCounted, 0)
			unknownMsgs := atomic.SwapUint64(&g.unknownMsgCount, 0)
			connBusy := atomic.SwapUint64(&g.connBusyCount, 0)

			if heartbeatTimeouts == 0 && loginTimeouts == 0 && loginLimited == 0 && unknownMsgs == 0 && connBusy == 0 {
				continue
			}
			g.logger.Info("gate stats",
				zap.Uint64("heartbeat_timeout", heartbeatTimeouts),
				zap.Uint64("login_timeout", loginTimeouts),
				zap.Uint64("login_rate_limited", loginLimited),
				zap.Uint64("unknown_msg", unknownMsgs),
				zap.Uint64("conn_busy", connBusy),
			)
		}
	}
}
