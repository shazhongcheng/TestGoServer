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

var (
	ErrPlayerClosed       = errors.New("player closed")
	ErrPlayerBusy         = errors.New("player inbox busy")
	ErrUnknownMessage     = errors.New("unknown player message")
	ErrPlayerDestroyed    = errors.New("player destroyed")
	ErrPlayerReplyTimeout = errors.New("player reply timeout")
)

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

	inbox chan Message
	state int32 // PlayerState

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

		state: int32(PlayerStateActive),
	}
	for _, m := range modules {
		_ = m.Init(p)
	}

	go p.loop()

	return p
}

// ================= lifecycle =================

func (p *Player) State() PlayerState {
	return PlayerState(atomic.LoadInt32(&p.state))
}

func (p *Player) OnResume(sessionID int64) {
	// 只允许 Offline -> Active
	if !atomic.CompareAndSwapInt32(
		&p.state,
		int32(PlayerStateOffline),
		int32(PlayerStateActive),
	) {
		return
	}

	p.SessionID = sessionID
	p.Context.SessionID = sessionID

	for _, m := range p.modules {
		m.OnResume()
	}
}

func (p *Player) OnOffline() {
	// Active -> Offline
	if !atomic.CompareAndSwapInt32(
		&p.state,
		int32(PlayerStateActive),
		int32(PlayerStateOffline),
	) {
		return
	}

	for _, m := range p.modules {
		m.OnOffline()
	}
}

// Destroy：只能被 PlayerManager 调用
func (p *Player) Destroy() {
	// Offline -> Destroyed
	if !atomic.CompareAndSwapInt32(
		&p.state,
		int32(PlayerStateOffline),
		int32(PlayerStateDestroyed),
	) {
		return
	}
	// loop 会自然退出
}

// ================= message =================

func (p *Player) Post(msg Message) error {
	st := p.State()
	if st == PlayerStateDestroyed {
		return ErrPlayerDestroyed
	}
	if st != PlayerStateActive {
		return ErrPlayerClosed
	}

	select {
	case p.inbox <- msg:
		return nil
	default:
		return ErrPlayerBusy
	}
}

func (p *Player) Notify(msgID int, env *internalpb.Envelope) error {
	return p.Post(Message{
		MsgID: msgID,
		Env:   env,
		Reply: nil,
	})
}

func (p *Player) Dispatch(
	msgID int,
	env *internalpb.Envelope,
) (*internalpb.Envelope, error) {

	reply := make(chan dispatchResult, 1)

	if err := p.Post(Message{
		MsgID: msgID,
		Env:   env,
		Reply: reply,
	}); err != nil {
		return nil, err
	}

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
		return nil, ErrPlayerReplyTimeout
	}
}

// ================= loop =================

//func (p *Player) loop() {
//	for {
//		// Destroyed 才真正退出
//		if p.State() == PlayerStateDestroyed {
//			return
//		}
//
//		select {
//		case msg := <-p.inbox:
//			p.handle(msg)
//		default:
//			time.Sleep(5 * time.Millisecond)
//		}
//	}
//}

func (p *Player) loop() {
	for {
		// Destroyed：清理并退出
		if p.State() == PlayerStateDestroyed {
			p.drainAndFail()
			return
		}

		msg := <-p.inbox
		p.handle(msg)
	}
}

func (p *Player) drainAndFail() {
	for {
		select {
		case msg := <-p.inbox:
			if msg.Reply != nil {
				msg.Reply <- dispatchResult{
					Envelope: nil,
					Handled:  false,
					Err:      ErrPlayerDestroyed,
				}
			}
		default:
			return
		}
	}
}

func (p *Player) handle(msg Message) {
	var rsp *internalpb.Envelope
	var handled bool
	var lastErr error
	replied := false

	defer func() {
		if r := recover(); r != nil {
			lastErr = fmt.Errorf("player panic: %v", r)
		}
		if msg.Reply != nil && !replied {
			msg.Reply <- dispatchResult{
				Envelope: rsp,
				Handled:  handled,
				Err:      lastErr,
			}
		}
	}()

	for _, m := range p.modules {
		if !m.CanHandle(msg.MsgID) {
			continue
		}

		func() {
			defer func() {
				if r := recover(); r != nil {
					lastErr = fmt.Errorf("module panic: %v", r)
					handled = false
				}
			}()
			rsp, handled, lastErr = m.Handle(msg.MsgID, msg.Env)
		}()

		if handled {
			break
		}
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
