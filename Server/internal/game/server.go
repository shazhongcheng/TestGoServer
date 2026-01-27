package game

import (
	"context"
	"log"
	"net"

	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/transport"
)

type Server struct {
	addr string
}

func NewServer(addr string) *Server {
	return &Server{addr: addr}
}

func (s *Server) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", s.addr)
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
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	for {
		env, err := transport.ReadEnvelope(conn)
		if err != nil {
			return
		}
		s.handleEnvelope(conn, env)
	}
}

func (s *Server) handleEnvelope(conn net.Conn, env *internalpb.Envelope) {
	switch int(env.MsgId) {
	case protocol.MsgPlayerEnterGameReq:
		log.Printf("[Game] player enter player=%d session=%d", env.PlayerId, env.SessionId)
		rsp := &internalpb.Envelope{
			MsgId:     protocol.MsgPlayerEnterGameRsp,
			SessionId: env.SessionId,
			PlayerId:  env.PlayerId,
		}
		if err := transport.WriteEnvelope(conn, rsp); err != nil {
			log.Printf("[Game] send enter rsp failed: %v", err)
		}
	default:
		log.Printf("[Game] unknown msgID=%d", env.MsgId)
	}
}
