package gate

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

type remoteClient struct {
	name       string
	addr       string
	logger     *zap.Logger
	onEnvelope func(env *internalpb.Envelope)

	mu   sync.RWMutex
	conn *transport.BufferedConn

	// ⭐ 发送队列（核心）
	sendCh chan *internalpb.Envelope
}

type remoteClientPool struct {
	clients []*remoteClient
}

func newRemoteClient(name, addr string, logger *zap.Logger, onEnvelope func(env *internalpb.Envelope)) *remoteClient {
	return &remoteClient{
		name:       name,
		addr:       addr,
		logger:     logger,
		onEnvelope: onEnvelope,

		// 队列大小可以根据压测调整
		sendCh: make(chan *internalpb.Envelope, 8192),
	}
}

func newRemoteClientPool(name, addr string, logger *zap.Logger, onEnvelope func(env *internalpb.Envelope), size int) *remoteClientPool {
	if size < 1 {
		size = 1
	}
	clients := make([]*remoteClient, 0, size)
	for i := 0; i < size; i++ {
		clients = append(clients, newRemoteClient(name, addr, logger, onEnvelope))
	}
	return &remoteClientPool{clients: clients}
}

func (p *remoteClientPool) Start(ctx context.Context) {
	for _, client := range p.clients {
		client.Start(ctx)
	}
}

func (p *remoteClientPool) Send(sessionID int64, env *internalpb.Envelope) error {
	if len(p.clients) == 0 {
		return protocol.InternalErrRemoteBusy
	}
	index := int(sessionID % int64(len(p.clients)))
	if index < 0 {
		index = -index
	}
	return p.clients[index].Send(env)
}

func (c *remoteClient) Start(ctx context.Context) {
	go c.connectLoop(ctx)
	go c.writeLoop(ctx)
}

func (c *remoteClient) Send(env *internalpb.Envelope) error {
	select {
	case c.sendCh <- env:
		return nil
	default:
		// 队列满 = 下游严重拥堵
		return protocol.InternalErrRemoteBusy
	}
}

// ========================
// Writer Loop（唯一写 socket 的地方）
// ========================
func (c *remoteClient) writeLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return

		case env := <-c.sendCh:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				// 远端未连接，直接丢弃 or 记录
				continue
			}

			if err := conn.WriteEnvelope(env); err != nil {
				c.logger.Warn("write to remote failed",
					zap.String("remote", c.name),
					zap.String("addr", c.addr),
					zap.Err("error", err),
				)
			}
		}
	}
}

func (c *remoteClient) connectLoop(ctx context.Context) {
	backoff := time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		conn, err := net.Dial("tcp", c.addr)
		if err != nil {
			c.logger.Warn("remote dial failed",
				zap.String("reason", err.Error()),
				zap.String("addr", c.addr),
				zap.String("remote", c.name),
				zap.Int64("session", 0),
				zap.Int64("player", 0),
				zap.Int("msg_id", 0),
				zap.Int64("conn_id", 0),
				zap.String("trace_id", ""),
			)
			time.Sleep(backoff)
			if backoff < 5*time.Second {
				backoff *= 2
			}
			continue
		}

		c.mu.Lock()
		c.conn = transport.NewBufferedConn(conn)
		c.mu.Unlock()
		c.logger.Info("remote connected",
			zap.String("addr", c.addr),
			zap.String("remote", c.name),
			zap.Int64("session", 0),
			zap.Int64("player", 0),
			zap.Int("msg_id", 0),
			zap.String("reason", ""),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		)
		backoff = time.Second

		for {
			env, err := c.conn.ReadEnvelope()
			if err != nil {
				break
			}
			if c.onEnvelope != nil {
				c.onEnvelope(env)
			}
		}

		_ = c.conn.Close()
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		c.logger.Warn("remote disconnected",
			zap.String("remote", c.name),
			zap.Int64("session", 0),
			zap.Int64("player", 0),
			zap.Int("msg_id", 0),
			zap.String("reason", ""),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		)
	}
}
