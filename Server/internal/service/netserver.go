package service

import (
	"context"
	"fmt"
	"net"
	"runtime"
	"sync"
	"sync/atomic"

	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/router"
	"game-server/internal/transport"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type NetServer struct {
	svc        *Server
	gameRouter *GameRouter

	mu         sync.RWMutex
	gateConns  map[int64]*transport.BufferedConn // gateID -> conn
	nextGateID int64
	// session -> gateID
	sessionGate map[int64]int64

	connOptions transport.ConnOptions

	routeMu     sync.RWMutex
	playerRoute map[int64]*GameRouter

	dispatchQueues []chan *internalpb.Envelope
	dispatchOnce   sync.Once
}

func NewNetServer(svc *Server, gameRouter *GameRouter, connOptions transport.ConnOptions) *NetServer {
	return &NetServer{
		svc:         svc,
		gameRouter:  gameRouter,
		gateConns:   make(map[int64]*transport.BufferedConn),
		sessionGate: make(map[int64]int64),
		playerRoute: make(map[int64]*GameRouter),
		connOptions: connOptions,
	}
}

func (n *NetServer) ListenAndServe(ctx context.Context, addr string) error {
	n.startDispatchers(ctx)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	defer ln.Close()

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				continue
			}
		}

		gateID := atomic.AddInt64(&n.nextGateID, 1)
		bc := transport.NewBufferedConnWithOptions(conn, n.connOptions)

		n.mu.Lock()
		n.gateConns[gateID] = bc
		n.mu.Unlock()

		n.svc.logger.Info("gate connected",
			zap.Int64("gate_id", gateID),
			zap.String("addr", conn.RemoteAddr().String()),
		)

		go n.handleGateConn(ctx, gateID, bc)
	}
}

func (n *NetServer) handleGateConn(ctx context.Context, gateID int64, conn *transport.BufferedConn) {
	defer func() {
		_ = conn.Close()

		n.mu.Lock()
		delete(n.gateConns, gateID)
		for sid, gid := range n.sessionGate {
			if gid == gateID {
				delete(n.sessionGate, sid)
			}
		}
		n.mu.Unlock()

		n.svc.logger.Warn("gate disconnected",
			zap.Int64("gate_id", gateID),
		)
	}()

	for {
		env, err := conn.ReadEnvelope()
		if err != nil {
			return
		}

		// 记录 session -> gate 映射
		if env.SessionId != 0 {
			n.mu.Lock()
			n.sessionGate[env.SessionId] = gateID
			n.mu.Unlock()
		}

		n.enqueueEnvelope(env)
	}
}

func (n *NetServer) dispatchEnvelope(ctx context.Context, env *internalpb.Envelope) {
	msgID := int(env.MsgId)
	if rule, ok := router.GetRoute(msgID); ok && rule.Target == router.TargetGame {
		n.routeToGame(env)
		return
	}

	serviceCtx := &Context{
		Context:   ctx,
		SessionID: env.SessionId,
		PlayerID:  env.PlayerId,
		MsgID:     msgID,
		Payload:   env.Payload,
		TraceID:   fmt.Sprintf("session-%d", env.SessionId),
		Reply: func(replyMsgID int, data []byte) error {
			return n.replyToGate(env.SessionId, replyMsgID, data)
		},
		Push: func(pushMsgID int, data []byte) error {
			return n.replyToGate(env.SessionId, pushMsgID, data)
		},
		// ⭐ 关键修正点
		ReplyError: nil, // 先占位

		SetPlayerID: func(playerID int64) {
			env.PlayerId = playerID
		},
		SendToGame: func(msgID int, data []byte) error {
			gameEnv := &internalpb.Envelope{
				MsgId:     int32(msgID),
				SessionId: env.SessionId,
				PlayerId:  env.PlayerId,
				Payload:   data,
			}
			return n.routeToGame(gameEnv)
		},
	}

	// ⭐ 在 Context 构造完成后，再绑定 ReplyError
	serviceCtx.ReplyError = makeReplyError(serviceCtx)

	n.svc.Handle(serviceCtx)
}

func (n *NetServer) startDispatchers(ctx context.Context) {
	n.dispatchOnce.Do(func() {
		workerCount := 4
		if cpuCount := runtime.GOMAXPROCS(0); cpuCount > 0 {
			workerCount = cpuCount * 2
		}
		n.dispatchQueues = make([]chan *internalpb.Envelope, workerCount)
		for i := 0; i < workerCount; i++ {
			queue := make(chan *internalpb.Envelope, 4096)
			n.dispatchQueues[i] = queue
			go func(ch <-chan *internalpb.Envelope) {
				for {
					select {
					case <-ctx.Done():
						return
					case env := <-ch:
						n.dispatchEnvelope(ctx, env)
					}
				}
			}(queue)
		}
	})
}

func (n *NetServer) enqueueEnvelope(env *internalpb.Envelope) {
	if len(n.dispatchQueues) == 0 {
		return
	}
	index := int(env.SessionId % int64(len(n.dispatchQueues)))
	if index < 0 {
		index = -index
	}
	n.dispatchQueues[index] <- env
}

func makeReplyError(ctx *Context) func(protocol.ErrorCode, string) error {
	return func(code protocol.ErrorCode, msg string) error {
		rsp := &internalpb.ErrorRsp{
			Code:    int32(code),
			Message: msg,
		}
		data, _ := proto.Marshal(rsp)
		return ctx.Reply(protocol.MsgErrorRsp, data)
	}
}

func (n *NetServer) replyToGate(sessionID int64, msgID int, data []byte) error {
	n.mu.RLock()
	gateID, ok := n.sessionGate[sessionID]
	if !ok {
		n.mu.RUnlock()
		return protocol.InternalErrNoGateConnection
	}
	conn := n.gateConns[gateID]
	n.mu.RUnlock()

	if conn == nil {
		return protocol.InternalErrNoGateConnection
	}

	env := &internalpb.Envelope{
		MsgId:     int32(msgID),
		SessionId: sessionID,
		Payload:   data,
	}
	return conn.WriteEnvelope(env)
}

func (n *NetServer) ForwardToGate(env *internalpb.Envelope) error {
	return n.replyToGate(env.SessionId, int(env.MsgId), env.Payload)
}

func (n *NetServer) routeToGame(env *internalpb.Envelope) error {
	if n.gameRouter == nil {
		return protocol.InternalErrRemoteNotReady
	}
	router := n.gameRouter
	msgID := int(env.MsgId)
	if env.PlayerId != 0 {
		n.routeMu.RLock()
		r, ok := n.playerRoute[env.PlayerId]
		n.routeMu.RUnlock()
		if ok {
			router = r
		} else {
			n.routeMu.Lock()
			n.playerRoute[env.PlayerId] = n.gameRouter
			n.routeMu.Unlock()
		}
	}
	err := router.Send(env)
	if env.PlayerId != 0 && msgID == protocol.MsgPlayerOfflineNotify {
		n.routeMu.Lock()
		delete(n.playerRoute, env.PlayerId)
		n.routeMu.Unlock()
	}
	return err
}
