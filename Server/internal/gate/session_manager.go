// internal/gate/session_manager.go
package gate

import (
	"sync"
	"time"
)

type SessionManager struct {
	sessions map[int64]*Session
	mu       sync.RWMutex
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*Session),
	}
}

func (sm *SessionManager) GC(timeout time.Duration) {
	now := time.Now()
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, s := range sm.sessions {
		if s.State == SessionOffline &&
			now.Sub(s.LastSeen) > timeout {
			delete(sm.sessions, id)
		}
	}
}
