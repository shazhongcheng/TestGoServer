// game/player/player.go
package player_module

import (
	"errors"
	"fmt"
	"time"

	//"game-server/internal/game"
	"game-server/internal/player_db"
	"game-server/internal/protocol/internalpb"
	"sync/atomic"
)

var ErrPlayerClosed = errors.New("player closed")
var ErrPlayerBusy = errors.New("player inbox busy")
var ErrUnknownMessage = errors.New("unknown player message")

type Message struct {
	MsgID int
	Env   *internalpb.Envelope
	Reply chan dispatchResult
}

type dispatchResult struct {
	Envelope *internalpb.Envelope
	Handled  bool
	Err      error
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
		replied := false
		handled := false
		var lastErr error

		func() {
			defer func() {
				if r := recover(); r != nil {
					// TODO: logger hook
					if msg.Reply != nil && !replied {
						msg.Reply <- dispatchResult{Envelope: nil, Handled: handled, Err: lastErr}
						replied = true
					}
				}
			}()

			for _, m := range p.modules {
				if !m.CanHandle(msg.MsgID) {
					continue
				}

				var err error
				func() {
					defer func() {
						if r := recover(); r != nil {
							err = fmt.Errorf("player module panic: %v", r)
							handled = false
						}
					}()
					rsp, handled, err = m.Handle(msg.MsgID, msg.Env)
				}()
				if err != nil {
					// TODO: logger hook
					lastErr = err
				}
				if handled {
					break
				}
			}
		}()

		if msg.Reply != nil && !replied {
			msg.Reply <- dispatchResult{Envelope: rsp, Handled: handled, Err: lastErr}
		}
	}
}

func (p *Player) Dispatch(
	msgID int,
	env *internalpb.Envelope,
) (*internalpb.Envelope, error) {

	if atomic.LoadInt32(&p.closed) == 1 {
		return nil, ErrPlayerClosed
	}

	reply := make(chan dispatchResult, 1)

	if err := p.Post(Message{
		MsgID: msgID,
		Env:   env,
		Reply: reply,
	}); err == nil {
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()

		select {
		case res := <-reply:
			if res.Err != nil {
				return nil, res.Err
			}
			if !res.Handled {
				return nil, ErrUnknownMessage
			}
			return res.Envelope, nil
		case <-timer.C:
			return nil, errors.New("player reply timeout")
		}
	}
	return nil, ErrPlayerBusy
}

func (p *Player) Notify(msgID int, env *internalpb.Envelope) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		return ErrPlayerClosed
	}

	return p.Post(Message{
		MsgID: msgID,
		Env:   env,
		Reply: nil, // 关键：没有 reply
	})
}

func (p *Player) Post(msg Message) error {
	if atomic.LoadInt32(&p.closed) == 1 {
		return ErrPlayerClosed
	}

	select {
	case p.inbox <- msg:
		return nil
	default:
		return ErrPlayerBusy
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
