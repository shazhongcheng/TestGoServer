package service

import (
	"context"
	"net"
	"sync"
	"time"

	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/transport"
	"go.uber.org/zap"
)

type GameRouter struct {
	addr string

	mu   sync.RWMutex
	conn *transport.BufferedConn

	sendCh chan *internalpb.Envelope
	closed chan struct{}

	logger         *zap.Logger
	connOptions    transport.ConnOptions
	sendRetryMax   int
	sendRetryDelay time.Duration
	busyCount      uint64
	dropCount      uint64
}

func NewGameRouter(addr string, logger *zap.Logger, options transport.ConnOptions, retryMax int, retryDelay time.Duration) *GameRouter {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &GameRouter{
		addr:           addr,
		logger:         logger,
		connOptions:    options,
		sendRetryMax:   retryMax,
		sendRetryDelay: retryDelay,
	}
}

func (r *GameRouter) Start(ctx context.Context, onEnvelope func(env *internalpb.Envelope)) {
	r.sendCh = make(chan *internalpb.Envelope, 2048)
	r.closed = make(chan struct{})

	go r.connectLoop(ctx, onEnvelope)
	go r.writeLoop()
}

func (r *GameRouter) Send(env *internalpb.Envelope) error {
	for attempt := 0; attempt <= r.sendRetryMax; attempt++ {
		select {
		case r.sendCh <- env:
			return nil
		default:
			if attempt == r.sendRetryMax {
				r.busyCount++
				r.logger.Warn("game router send queue full",
					zap.String("addr", r.addr),
					zap.Int("msg_id", int(env.MsgId)),
					zap.Int64("session", env.SessionId),
					zap.Int64("player", env.PlayerId),
					zap.String("reason", "send_queue_full"),
				)
				return protocol.InternalErrGameRouterBusy
			}
			time.Sleep(r.sendRetryDelay)
		}
	}
	return protocol.InternalErrGameRouterBusy
}

func (r *GameRouter) writeLoop() {
	for {
		select {
		case env := <-r.sendCh:
			r.mu.RLock()
			conn := r.conn
			r.mu.RUnlock()

			if conn == nil {
				r.dropCount++
				r.logger.Warn("game router disconnected, drop message",
					zap.String("addr", r.addr),
					zap.Int("msg_id", int(env.MsgId)),
					zap.Int64("session", env.SessionId),
					zap.Int64("player", env.PlayerId),
					zap.String("reason", "router_disconnected"),
				)
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

		dialer := net.Dialer{Timeout: 3 * time.Second, KeepAlive: r.connOptions.KeepAlive}
		conn, err := dialer.Dial("tcp", r.addr)
		if err != nil {
			time.Sleep(backoff)
			if backoff < 5*time.Second {
				backoff *= 2
			}
			continue
		}

		r.mu.Lock()
		r.conn = transport.NewBufferedConnWithOptions(conn, r.connOptions)
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
