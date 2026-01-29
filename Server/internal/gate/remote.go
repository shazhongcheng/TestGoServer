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

	connOptions      transport.ConnOptions
	sendRetryMax     int
	sendRetryBackoff time.Duration
	busyCount        uint64
	dropCount        uint64
	lastConnectedAt  time.Time
}

type remoteClientPool struct {
	clients []*remoteClient
}

func newRemoteClient(name, addr string, logger *zap.Logger, onEnvelope func(env *internalpb.Envelope), options transport.ConnOptions, retryMax int, retryBackoff time.Duration) *remoteClient {
	return &remoteClient{
		name:       name,
		addr:       addr,
		logger:     logger,
		onEnvelope: onEnvelope,

		// 队列大小可以根据压测调整
		sendCh:           make(chan *internalpb.Envelope, 8192),
		connOptions:      options,
		sendRetryMax:     retryMax,
		sendRetryBackoff: retryBackoff,
	}
}

func newRemoteClientPool(name, addr string, logger *zap.Logger, onEnvelope func(env *internalpb.Envelope), size int, options transport.ConnOptions, retryMax int, retryBackoff time.Duration) *remoteClientPool {
	if size < 1 {
		size = 1
	}
	clients := make([]*remoteClient, 0, size)
	for i := 0; i < size; i++ {
		clients = append(clients, newRemoteClient(name, addr, logger, onEnvelope, options, retryMax, retryBackoff))
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
	go c.heartbeatLoop(ctx)
}

func (c *remoteClient) Send(env *internalpb.Envelope) error {
	for attempt := 0; attempt <= c.sendRetryMax; attempt++ {
		select {
		case c.sendCh <- env:
			return nil
		default:
			if attempt == c.sendRetryMax {
				c.busyCount++
				c.logger.Warn("remote send queue full",
					zap.String("remote", c.name),
					zap.String("addr", c.addr),
					zap.Int("msg_id", int(env.MsgId)),
					zap.Int64("session", env.SessionId),
					zap.Int64("player", env.PlayerId),
					zap.String("reason", "send_queue_full"),
				)
				return protocol.InternalErrRemoteBusy
			}
			time.Sleep(c.sendRetryBackoff)
		}
	}
	return protocol.InternalErrRemoteBusy
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
				c.dropCount++
				c.logger.Warn("remote disconnected, drop message",
					zap.String("remote", c.name),
					zap.String("addr", c.addr),
					zap.Int("msg_id", int(env.MsgId)),
					zap.Int64("session", env.SessionId),
					zap.Int64("player", env.PlayerId),
					zap.String("reason", "remote_disconnected"),
				)
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

		dialer := net.Dialer{Timeout: 3 * time.Second, KeepAlive: c.connOptions.KeepAlive}
		conn, err := dialer.Dial("tcp", c.addr)
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
		c.conn = transport.NewBufferedConnWithOptions(conn, c.connOptions)
		c.mu.Unlock()
		c.lastConnectedAt = time.Now()
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
			zap.Duration("connected_duration", time.Since(c.lastConnectedAt)),
			zap.Int64("session", 0),
			zap.Int64("player", 0),
			zap.Int("msg_id", 0),
			zap.String("reason", ""),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		)
	}
}

func (c *remoteClient) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.mu.RLock()
			conn := c.conn
			c.mu.RUnlock()

			if conn == nil {
				continue
			}

			_ = conn.WriteEnvelope(&internalpb.Envelope{
				MsgId: protocol.MsgServicePing,
			})
		}
	}
}
