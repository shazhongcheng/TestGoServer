package login

import (
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"google.golang.org/protobuf/proto"

	"game-server/internal/service"
)

func (m *Module) Handlers() map[int]service.HandlerFunc {
	return map[int]service.HandlerFunc{
		protocol.MsgLoginReq: m.onLogin,
	}
}

func (m *Module) onLogin(ctx *service.Context) error {
	var req internalpb.LoginReq
	if err := proto.Unmarshal(ctx.Payload, &req); err != nil {
		return err
	}

	// 🚨 这里先 mock，一个固定 playerID
	playerID := int64(10001)

	rsp := &internalpb.LoginRsp{
		PlayerId: playerID,
	}
	data, _ := proto.Marshal(rsp)

	// 回包给 Gate → Client
	return ctx.Reply(protocol.MsgLoginRsp, data)
}
