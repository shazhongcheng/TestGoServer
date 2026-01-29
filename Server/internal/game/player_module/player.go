// game/player/player.go
package player_module

import (
	"errors"
	//"game-server/internal/game"
	"game-server/internal/player_db"
	"game-server/internal/protocol/internalpb"
	"sync/atomic"
)

var ErrPlayerClosed = errors.New("player closed")

type Message struct {
	MsgID int
	Env   *internalpb.Envelope
	Reply chan *internalpb.Envelope
}

type Player struct {
	//mu sync.Mutex

	PlayerID  int64
	SessionID int64

	Context PlayerContext
	Profile player_db.PlayerProfile

	inbox  chan Message
	closed int32

	modules []Module
}

func NewPlayer(playerID, sessionID int64, profile player_db.PlayerProfile, modules []Module) *Player {
	p := &Player{
		PlayerID:  playerID,
		SessionID: sessionID,

		Context: PlayerContext{
			PlayerID:  playerID,
			SessionID: sessionID,
		},

		Profile: profile,
		modules: modules,

		inbox: make(chan Message, 64),
	}
	for _, m := range modules {
		_ = m.Init(p)
	}

	go p.loop()

	return p
}

func (p *Player) loop() {
	for msg := range p.inbox {
		var rsp *internalpb.Envelope

		for _, m := range p.modules {
			if !m.CanHandle(msg.MsgID) {
				continue
			}

			var handled bool
			var err error
			rsp, handled, err = m.Handle(msg.MsgID, msg.Env)
			if err != nil {
				// TODO: logger hook
			}
			if handled {
				break
			}
		}

		if msg.Reply != nil {
			msg.Reply <- rsp
		}
	}
}

func (p *Player) Dispatch(
	msgID int,
	env *internalpb.Envelope,
) (*internalpb.Envelope, error) {

	if atomic.LoadInt32(&p.closed) == 1 {
		return nil, nil
	}

	reply := make(chan *internalpb.Envelope, 1)

	p.inbox <- Message{
		MsgID: msgID,
		Env:   env,
		Reply: reply,
	}

	select {
	case p.inbox <- Message{
		MsgID: msgID,
		Env:   env,
		Reply: reply,
	}:
		return <-reply, nil
	default:
		// inbox full = backpressure
		return nil, ErrPlayerClosed
	}
}

//func (p *Player) Dispatch(ctx context.Context, msgID int, env *internalpb.Envelope) (*internalpb.Envelope, error) {
//	for _, m := range p.modules {
//		if rsp, ok, err := m.Handle(ctx, msgID, env); ok {
//			return rsp, err
//		}
//	}
//	return nil, nil
//}

func (p *Player) OnResume(sessionID int64) {
	if atomic.LoadInt32(&p.closed) == 1 {
		return
	}

	p.SessionID = sessionID
	p.Context.SessionID = sessionID

	for _, m := range p.modules {
		m.OnResume()
	}
}

func (p *Player) OnOffline() {
	if !atomic.CompareAndSwapInt32(&p.closed, 0, 1) {
		return
	}

	for _, m := range p.modules {
		m.OnOffline()
	}

	close(p.inbox)
}

//func (p *Player) SetSession(sessionID int64) {
//	p.mu.Lock()
//	p.SessionID = sessionID
//	p.Context.SessionID = sessionID
//	p.mu.Unlock()
//}

func (p *Player) ToPlayerData() *internalpb.PlayerData {
	return &internalpb.PlayerData{
		RoleId:   p.Profile.RoleID,
		Nickname: p.Profile.NickName,
		Level:    p.Profile.Level,
		Exp:      p.Profile.Exp,
		Gold:     p.Profile.Gold,
		Stamina:  p.Profile.Stamina,
	}
}
