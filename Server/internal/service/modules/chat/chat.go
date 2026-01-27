// internal/service/modules/chat/chat.go
package chat

import "game-server/internal/service"

const (
	MsgChatSend = 2001
)

type Module struct{}

func (m *Module) Name() string { return "chat" }
func (m *Module) Init() error  { return nil }

func (m *Module) Handlers() map[int]service.HandlerFunc {
	return map[int]service.HandlerFunc{
		MsgChatSend: m.onChat,
	}
}

func (m *Module) onChat(ctx *service.Context) error {
	// 广播 / 跨服推送
	return nil
}
