package login

import (
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/service"

	"google.golang.org/protobuf/proto"
)

func (m *Module) Handlers() map[int]service.HandlerFunc {
	return map[int]service.HandlerFunc{
		protocol.MsgLoginReq: m.onLogin,
	}
}

func (m *Module) verifyToken(platform int32, token string) error {
	if token == "" {
		return protocol.InternalErrInvalidToken
	}

	switch platform {

	case protocol.PlatformTest:
		// 测试平台，永远放行
		return nil

	//case protocol.PlatformAndroid:
	//	return m.verifyAndroidToken(token)
	//
	//case protocol.PlatformIOS:
	//	return m.verifyIOSToken(token)
	//
	//case protocol.PlatformPC:
	//	return m.verifyPCToken(token)

	default:
		return protocol.InternalErrUnknownPlatForm
	}
}
func (m *Module) onLogin(ctx *service.Context) error {
	var req internalpb.LoginReq
	if err := proto.Unmarshal(ctx.Payload, &req); err != nil {
		return err
	}

	switch req.Platform {
	case protocol.PlatformTest:
		// 不校验
	case protocol.PlatformAndroid,
		protocol.PlatformIOS,
		protocol.PlatformPC:

		if err := m.verifyToken(req.Platform, req.Token); err != nil {
			return ctx.ReplyError(
				protocol.ErrInvalidToken,
				err.Error(),
			)
		}
	default:
		return ctx.ReplyError(
			protocol.ErrUnknownPlatform,
			protocol.InternalErrUnknownPlatForm.Error(),
		)
	}

	playerID, err := m.svc.NextUID(ctx)
	if err != nil {
		return err
	}

	rsp := &internalpb.LoginRsp{
		PlayerId: playerID,
	}
	data, _ := proto.Marshal(rsp)

	ctx.SetPlayerID(playerID)

	if ctx.SendToGame != nil {
		_ = ctx.SendToGame(protocol.MsgPlayerEnterGameReq, nil)
	}

	// 回包给 Gate → Client
	return ctx.Reply(protocol.MsgLoginRsp, data)
}
