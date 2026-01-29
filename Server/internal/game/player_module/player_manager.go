// game/player/manager.go
package player_module

import (
	"context"
	"game-server/internal/player_db"
	"sync"
)

type PlayerManager struct {
	mu       sync.RWMutex
	players  map[int64]*Player
	sessions map[int64]int64
	store    player_db.Store
}

func NewPlayerManager(store player_db.Store) *PlayerManager {
	return &PlayerManager{
		players:  make(map[int64]*Player),
		sessions: make(map[int64]int64),
		store:    store,
	}
}

func (m *PlayerManager) GetOrCreate(ctx context.Context, sessionID, playerID int64) (*Player, error) {
	m.mu.RLock()
	p := m.players[playerID]
	m.mu.RUnlock()
	if p != nil {
		p.SessionID = sessionID
		p.OnResume(sessionID)
		m.mu.Lock()
		m.sessions[sessionID] = p.PlayerID
		m.mu.Unlock()
		return p, nil
	}

	profile, _, err := m.store.LoadProfile(ctx, playerID)
	if err != nil {
		return nil, err
	}
	if profile == nil {
		tmp := player_db.NewProfile(playerID, "")
		profile = &tmp
		if err := m.store.SaveProfile(ctx, profile); err != nil {
			return nil, err
		}
	}

	p = NewPlayer(playerID, sessionID, *profile, CreateModules())

	m.mu.Lock()
	m.players[playerID] = p
	m.sessions[sessionID] = playerID
	m.mu.Unlock()
	return p, nil
}

func (m *PlayerManager) MarkOffline(playerID int64) {
	m.mu.RLock()
	p := m.players[playerID]
	m.mu.RUnlock()
	if p != nil {
		_ = m.store.SaveProfile(context.Background(), &p.Profile)
		p.OnOffline()
		m.mu.Lock()
		delete(m.sessions, p.SessionID)
		m.mu.Unlock()
	}
}

func (m *PlayerManager) GetBySessionID(sessionID int64) *Player {
	m.mu.RLock()
	playerID := m.sessions[sessionID]
	p := m.players[playerID]
	m.mu.RUnlock()
	return p
}

func (m *PlayerManager) SaveAll(ctx context.Context) {
	m.mu.RLock()
	players := make([]*Player, 0, len(m.players))
	for _, p := range m.players {
		players = append(players, p)
	}
	m.mu.RUnlock()
	for _, p := range players {
		_ = m.store.SaveProfile(ctx, &p.Profile)
	}
}
