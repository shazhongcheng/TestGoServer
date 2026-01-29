// game/player/module.go
package player_module

import (
	"game-server/internal/protocol/internalpb"
)

type Module interface {
	Name() string

	// 生命周期
	Init(p *Player) error // Player 创建时
	OnResume()            // 断线重连
	OnOffline()           // 下线

	// 消息
	CanHandle(msgID int) bool
	Handle(
		msgID int,
		env *internalpb.Envelope,
	) (*internalpb.Envelope, bool, error)
}
