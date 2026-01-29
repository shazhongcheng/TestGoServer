// game/player/modules/base/base.go
package base

import (
	"game-server/internal/game/player_module"

	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"google.golang.org/protobuf/proto"
)

type BaseModule struct {
	p *player_module.Player
}

func New() player_module.Module {
	return &BaseModule{}
}

func (m *BaseModule) Name() string { return "base" }

func (m *BaseModule) CanHandle(msgID int) bool {
	return msgID == protocol.MsgPlayerEnterGameReq ||
		msgID == protocol.MsgLoadPlayerDataReq
}

func (m *BaseModule) Init(p *player_module.Player) error {
	m.p = p
	return nil
}

func (m *BaseModule) Handle(
	msgID int,
	env *internalpb.Envelope,
) (*internalpb.Envelope, bool, error) {

	switch msgID {

	case protocol.MsgPlayerEnterGameReq:
		rsp := &internalpb.PlayerInitRsp{
			Data: m.p.ToPlayerData(),
		}
		data, _ := proto.Marshal(rsp)
		return &internalpb.Envelope{
			MsgId:     protocol.MsgPlayerEnterGameRsp,
			SessionId: env.SessionId,
			PlayerId:  env.PlayerId,
			Payload:   data,
		}, true, nil

	case protocol.MsgLoadPlayerDataReq:
		rsp := &internalpb.LoadPlayerDataRsp{
			Data: m.p.ToPlayerData(),
		}
		data, _ := proto.Marshal(rsp)
		return &internalpb.Envelope{
			MsgId:     protocol.MsgLoadPlayerDataRsp,
			SessionId: env.SessionId,
			PlayerId:  env.PlayerId,
			Payload:   data,
		}, true, nil
	}

	return nil, false, nil
}

func (m *BaseModule) OnResume() {
	// resume hook
}

func (m *BaseModule) OnOffline() {
	// offline hook
}
