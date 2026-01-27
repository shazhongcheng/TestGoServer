package game

import (
	"context"
	"fmt"
	"sync"

	"game-server/internal/player"
)

type PlayerManager struct {
	mu      sync.RWMutex
	players map[int64]*PlayerInfo
	store   player.Store
}

func NewPlayerManager(store player.Store) *PlayerManager {
	return &PlayerManager{
		players: make(map[int64]*PlayerInfo),
		store:   store,
	}
}

func (m *PlayerManager) EnsurePlayer(ctx context.Context, sessionID, playerID int64) (*PlayerInfo, error) {
	if playerID == 0 {
		return nil, fmt.Errorf("player id empty")
	}

	m.mu.RLock()
	info := m.players[playerID]
	m.mu.RUnlock()
	if info != nil {
		info.SetSession(sessionID)
		return info, nil
	}

	profile := player.NewProfile(playerID, "")
	if m.store != nil {
		if existing, ok, err := m.store.LoadProfile(ctx, playerID); err != nil {
			return nil, err
		} else if ok {
			profile = *existing
		} else if err := m.store.SaveProfile(ctx, &profile); err != nil {
			return nil, err
		}
	}

	info = &PlayerInfo{
		Context: PlayerContext{
			PlayerID:  playerID,
			SessionID: sessionID,
		},
		Profile: profile,
	}

	m.mu.Lock()
	m.players[playerID] = info
	m.mu.Unlock()
	return info, nil
}

func (m *PlayerManager) ResumePlayer(ctx context.Context, sessionID, playerID int64) (*PlayerInfo, error) {
	return m.EnsurePlayer(ctx, sessionID, playerID)
}

func (m *PlayerManager) MarkOffline(playerID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if info, ok := m.players[playerID]; ok {
		info.SetSession(0)
	}
}
