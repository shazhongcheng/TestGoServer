package gate

import "time"

func (g *Gate) allowLogin(s *Session) bool {
	if s == nil || g.loginRateLimitCount <= 0 {
		return true
	}
	now := time.Now()
	if s.LoginWindowStart.IsZero() || now.Sub(s.LoginWindowStart) > g.loginRateLimitWindow {
		s.LoginWindowStart = now
		s.LoginAttempts = 0
	}
	s.LoginAttempts++
	return s.LoginAttempts <= g.loginRateLimitCount
}
