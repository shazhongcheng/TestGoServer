package service

import (
	"context"
	"net"
	"sync"
	"time"

	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/transport"
)

type GameRouter struct {
	addr string

	mu   sync.RWMutex
	conn *transport.BufferedConn

	sendCh chan *internalpb.Envelope
	closed chan struct{}
}

func NewGameRouter(addr string) *GameRouter {
	return &GameRouter{addr: addr}
}

func (r *GameRouter) Start(ctx context.Context, onEnvelope func(env *internalpb.Envelope)) {
	r.sendCh = make(chan *internalpb.Envelope, 2048)
	r.closed = make(chan struct{})

	go r.connectLoop(ctx, onEnvelope)
	go r.writeLoop()
}

func (r *GameRouter) Send(env *internalpb.Envelope) error {
	select {
	case r.sendCh <- env:
		return nil
	default:
		return protocol.InternalErrGameRouterBusy
	}
}

func (r *GameRouter) writeLoop() {
	for {
		select {
		case env := <-r.sendCh:
			r.mu.RLock()
			conn := r.conn
			r.mu.RUnlock()

			if conn == nil {
				continue
			}

			if err := conn.WriteEnvelope(env); err != nil {
				// 写失败，等待重连
				time.Sleep(10 * time.Millisecond)
			}

		case <-r.closed:
			return
		}
	}
}

func (r *GameRouter) connectLoop(ctx context.Context, onEnvelope func(env *internalpb.Envelope)) {
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := net.Dial("tcp", r.addr)
		if err != nil {
			time.Sleep(backoff)
			if backoff < 5*time.Second {
				backoff *= 2
			}
			continue
		}

		r.mu.Lock()
		r.conn = transport.NewBufferedConn(conn)
		r.mu.Unlock()
		backoff = time.Second

		for {
			env, err := r.conn.ReadEnvelope()
			if err != nil {
				break
			}
			if onEnvelope != nil {
				onEnvelope(env)
			}
		}

		_ = r.conn.Close()
		r.mu.Lock()
		r.conn = nil
		r.mu.Unlock()
	}
}
