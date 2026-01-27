package gate

import (
	"context"
	"game-server/internal/protocol"
	"net"
	"sync"
	"time"

	"game-server/internal/common/selflog"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/transport"
)

type remoteClient struct {
	name       string
	addr       string
	logger     selflog.Logger
	onEnvelope func(env *internalpb.Envelope)

	mu   sync.RWMutex
	conn net.Conn
}

func newRemoteClient(name, addr string, logger selflog.Logger, onEnvelope func(env *internalpb.Envelope)) *remoteClient {
	return &remoteClient{
		name:       name,
		addr:       addr,
		logger:     logger,
		onEnvelope: onEnvelope,
	}
}

func (c *remoteClient) Start(ctx context.Context) {
	go c.connectLoop(ctx)
}

func (c *remoteClient) Send(env *internalpb.Envelope) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return protocol.InternalErrRemoteNotReady
	}
	return transport.WriteEnvelope(conn, env)
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
			c.logger.Warn("[%s] dial %s failed: %v", c.name, c.addr, err)
			time.Sleep(backoff)
			if backoff < 5*time.Second {
				backoff *= 2
			}
			continue
		}

		c.mu.Lock()
		c.conn = conn
		c.mu.Unlock()
		c.logger.Info("[%s] connected %s", c.name, c.addr)
		backoff = time.Second

		for {
			env, err := transport.ReadEnvelope(conn)
			if err != nil {
				break
			}
			if c.onEnvelope != nil {
				c.onEnvelope(env)
			}
		}

		_ = conn.Close()
		c.mu.Lock()
		c.conn = nil
		c.mu.Unlock()
		c.logger.Warn("[%s] disconnected", c.name)
	}
}
