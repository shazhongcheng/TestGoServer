package service

import (
	"context"
	"game-server/internal/protocol"
	"net"
	"sync"
	"time"

	"game-server/internal/protocol/internalpb"
	"game-server/internal/transport"
)

type GameRouter struct {
	addr string

	mu   sync.RWMutex
	conn net.Conn
}

func NewGameRouter(addr string) *GameRouter {
	return &GameRouter{addr: addr}
}

func (r *GameRouter) Start(ctx context.Context, onEnvelope func(env *internalpb.Envelope)) {
	go r.connectLoop(ctx, onEnvelope)
}

func (r *GameRouter) Send(env *internalpb.Envelope) error {
	r.mu.RLock()
	conn := r.conn
	r.mu.RUnlock()
	if conn == nil {
		return protocol.InternalErrGameRouterNotReady
	}
	return transport.WriteEnvelope(conn, env)
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
		r.conn = conn
		r.mu.Unlock()
		backoff = time.Second

		for {
			env, err := transport.ReadEnvelope(conn)
			if err != nil {
				break
			}
			if onEnvelope != nil {
				onEnvelope(env)
			}
		}

		_ = conn.Close()
		r.mu.Lock()
		r.conn = nil
		r.mu.Unlock()
	}
}
