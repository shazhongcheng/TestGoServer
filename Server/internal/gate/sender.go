// internal/gate/sender.go
package gate

import "game-server/internal/protocol/internalpb"

func (g *Gate) sendToService(module string, env *internalpb.Envelope) {
	// 这里可以是：
	// TCP
	// gRPC
	// 本地 channel
}

func (g *Gate) sendToGame(env *internalpb.Envelope) {
}
