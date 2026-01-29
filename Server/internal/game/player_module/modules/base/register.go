package base

import "game-server/internal/game/player_module"

func init() {
	player_module.RegisterModule(New)
}
