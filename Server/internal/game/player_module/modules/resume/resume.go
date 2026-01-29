// game/player/player_module/modules/resume.go
package resume

import (
	"game-server/internal/game/player_module"
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
)

type ResumeModule struct {
	p *player_module.Player
}

func (m *ResumeModule) Name() string {
	return "resume"
}

func (m *ResumeModule) OnResume() {

}

func (m *ResumeModule) OnOffline() {

}

func New() player_module.Module {
	return &ResumeModule{}
}

func (m *ResumeModule) Init(p *player_module.Player) error {
	m.p = p
	return nil
}

func (m *ResumeModule) CanHandle(msgID int) bool {
	return msgID == protocol.MsgPlayerResumeReq
}

func (m *ResumeModule) Handle(msgID int, env *internalpb.Envelope) (*internalpb.Envelope, bool, error) {
	// 显式 Resume
	m.p.OnResume(env.SessionId)
	return nil, true, nil
}
