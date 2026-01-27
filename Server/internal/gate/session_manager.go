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

func (sm *SessionManager) Add(s *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.sessions[s.ID] = s
}

func (sm *SessionManager) Get(id int64) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.sessions[id]
}

func (sm *SessionManager) Remove(id int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.sessions, id)
}

func (sm *SessionManager) snapshot() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	items := make([]*Session, 0, len(sm.sessions))
	for _, s := range sm.sessions {
		items = append(items, s)
	}
	return items
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
