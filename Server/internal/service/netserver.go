package service

import (
	"context"
	"fmt"
	"net"
	"sync"

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

	mu          sync.RWMutex
	gateConn    *transport.BufferedConn
	connOptions transport.ConnOptions

	routeMu     sync.RWMutex
	playerRoute map[int64]*GameRouter
}

func NewNetServer(svc *Server, gameRouter *GameRouter, connOptions transport.ConnOptions) *NetServer {
	return &NetServer{
		svc:         svc,
		gameRouter:  gameRouter,
		playerRoute: make(map[int64]*GameRouter),
		connOptions: connOptions,
	}
}

func (n *NetServer) ListenAndServe(ctx context.Context, addr string) error {
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
		n.mu.Lock()
		if n.gateConn != nil {
			n.mu.Unlock()
			n.svc.logger.Warn("reject gate connection",
				zap.String("reason", "single_gate_only"),
				zap.String("addr", conn.RemoteAddr().String()),
			)
			_ = conn.Close()
			continue
		}
		n.gateConn = transport.NewBufferedConnWithOptions(conn, n.connOptions)
		n.mu.Unlock()
		go n.handleGateConn(ctx)
	}
}

func (n *NetServer) handleGateConn(ctx context.Context) {
	n.mu.RLock()
	conn := n.gateConn
	n.mu.RUnlock()
	if conn == nil {
		return
	}
	defer conn.Close()
	for {
		env, err := conn.ReadEnvelope()
		if err != nil {
			n.svc.logger.Warn("gate connection closed",
				zap.String("reason", err.Error()),
			)
			n.mu.Lock()
			if n.gateConn == conn {
				n.gateConn = nil
			}
			n.mu.Unlock()
			return
		}
		n.dispatchEnvelope(ctx, env)
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
	conn := n.gateConn
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
