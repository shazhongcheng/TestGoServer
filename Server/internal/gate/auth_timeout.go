// internal/gate/auth_timeout.go
package gate

import "time"

func (g *Gate) checkAuthingTimeout() {
	now := time.Now()

	for _, s := range g.sessions.snapshot() {
		if s.State != SessionAuthing {
			continue
		}

		if now.Sub(s.AuthStart) > g.loginTimeout {
			g.logger.Warn(
				"login timeout session=%d",
				s.ID,
			)
			g.Kick(s.ID, "login timeout")
		}
	}
}
