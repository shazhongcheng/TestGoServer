// internal/service/context.go
package service

import "context"

type Context struct {
	context.Context

	SessionID int64
	PlayerID  int64
	MsgID     int
	Payload   []byte

	// 回包 / 推送
	Reply func(msgID int, data []byte) error
	Push  func(msgID int, data []byte) error
}
