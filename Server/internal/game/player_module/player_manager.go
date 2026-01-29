// game/player/manager.go
package player_module

import (
	"context"
	"game-server/internal/player_db"
	"sync"
)

type PlayerManager struct {
	mu      sync.RWMutex
	players map[int64]*Player
	store   player_db.Store
}

func NewPlayerManager(store player_db.Store) *PlayerManager {
	return &PlayerManager{
		players: make(map[int64]*Player),
		store:   store,
	}
}

func (m *PlayerManager) GetOrCreate(ctx context.Context, sessionID, playerID int64) (*Player, error) {
	m.mu.RLock()
	p := m.players[playerID]
	m.mu.RUnlock()
	if p != nil {
		p.SessionID = sessionID
		p.OnResume(sessionID)
		return p, nil
	}

	profile, _, _ := m.store.LoadProfile(ctx, playerID)
	if profile == nil {
		tmp := player_db.NewProfile(playerID, "")
		profile = &tmp
		_ = m.store.SaveProfile(ctx, profile)
	}

	p = NewPlayer(playerID, sessionID, *profile, CreateModules())

	m.mu.Lock()
	m.players[playerID] = p
	m.mu.Unlock()
	return p, nil
}

func (m *PlayerManager) MarkOffline(playerID int64) {
	m.mu.RLock()
	p := m.players[playerID]
	m.mu.RUnlock()
	if p != nil {
		p.OnOffline()
	}
}
