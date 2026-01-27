// internal/service/context.go
package service

import (
	"context"
	"game-server/internal/protocol"
)

type Context struct {
	context.Context

	SessionID int64
	PlayerID  int64
	MsgID     int
	Payload   []byte

	// 回包 / 推送
	Reply      func(msgID int, data []byte) error
	Push       func(msgID int, data []byte) error
	ReplyError func(code protocol.ErrorCode, msg string) error
	// 更新 Gate 会话信息
	SetPlayerID func(playerID int64)
	// 转发到 Game
	SendToGame func(msgID int, data []byte) error
}
