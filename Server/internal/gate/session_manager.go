// internal/gate/session_manager.go
package gate

import (
	"sync"
	"time"
)

type SessionManager struct {
	mu sync.RWMutex

	bySession map[int64]*Session
	byPlayer  map[int64]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		bySession: make(map[int64]*Session),
		byPlayer:  make(map[int64]*Session),
	}
}

func (sm *SessionManager) Add(s *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.bySession[s.ID] = s
}

func (sm *SessionManager) Get(sessionID int64) *Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return sm.bySession[sessionID]
}

func (sm *SessionManager) Remove(sessionID int64) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	s := sm.bySession[sessionID]
	if s == nil {
		return
	}

	// 清理 player 索引
	if s.PlayerID != 0 {
		if cur := sm.byPlayer[s.PlayerID]; cur == s {
			delete(sm.byPlayer, s.PlayerID)
		}
	}

	delete(sm.bySession, sessionID)
}

func (sm *SessionManager) snapshot() []*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	items := make([]*Session, 0, len(sm.bySession))
	for _, s := range sm.bySession {
		items = append(items, s)
	}
	return items
}

func (sm *SessionManager) GC(timeout time.Duration) []*Session {
	now := time.Now()
	var removed []*Session

	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, s := range sm.bySession {
		if s.State == SessionOffline &&
			now.Sub(s.LastSeen) > timeout {

			s.State = SessionClosed
			removed = append(removed, s)

			if s.PlayerID != 0 {
				delete(sm.byPlayer, s.PlayerID)
			}
			delete(sm.bySession, id)
		}
	}

	return removed
}

func (sm *SessionManager) BindPlayer(s *Session, playerID int64) (old *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	old = sm.byPlayer[playerID]

	s.PlayerID = playerID
	sm.byPlayer[playerID] = s

	return old
}
