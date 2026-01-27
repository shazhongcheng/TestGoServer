package game

import (
	"game-server/internal/player"
	"game-server/internal/protocol/internalpb"
)

type PlayerInfo struct {
	Context PlayerContext
	Profile player.PlayerProfile
}

func (p *PlayerInfo) SetSession(sessionID int64) {
	p.Context.SessionID = sessionID
}

func (p *PlayerInfo) ToPlayerData() *internalpb.PlayerData {
	return &internalpb.PlayerData{
		RoleId:   p.Profile.RoleID,
		Nickname: p.Profile.NickName,
		Level:    p.Profile.Level,
		Exp:      p.Profile.Exp,
		Gold:     p.Profile.Gold,
		Stamina:  p.Profile.Stamina,
	}
}
