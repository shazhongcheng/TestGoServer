package game

import (
	"context"
	"log"
	"net"

	"game-server/internal/player"
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/transport"

	"google.golang.org/protobuf/proto"
)

type Server struct {
	addr    string
	players *PlayerManager
}

func NewServer(addr string, store player.Store) *Server {
	return &Server{
		addr:    addr,
		players: NewPlayerManager(store),
	}
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
	log.Printf("[Game] player msgId=%d session=%d PlayerId=%d", env.MsgId, env.SessionId, env.PlayerId)
	switch int(env.MsgId) {
	case protocol.MsgPlayerEnterGameReq:
		log.Printf("[Game] player enter player=%d session=%d", env.PlayerId, env.SessionId)
		info, err := s.players.EnsurePlayer(context.Background(), env.SessionId, env.PlayerId)
		if err != nil {
			log.Printf("[Game] load player failed: %v", err)
			return
		}
		initRsp := &internalpb.PlayerInitRsp{
			Data: info.ToPlayerData(),
		}
		payload, _ := proto.Marshal(initRsp)
		rsp := &internalpb.Envelope{
			MsgId:     protocol.MsgPlayerEnterGameRsp,
			SessionId: env.SessionId,
			PlayerId:  env.PlayerId,
			Payload:   payload,
		}
		if err := transport.WriteEnvelope(conn, rsp); err != nil {
			log.Printf("[Game] send enter rsp failed: %v", err)
		}
	case protocol.MsgLoadPlayerDataReq:
		info, err := s.players.EnsurePlayer(context.Background(), env.SessionId, env.PlayerId)
		if err != nil {
			log.Printf("[Game] load player data failed: %v", err)
			return
		}
		dataRsp := &internalpb.LoadPlayerDataRsp{
			Data: info.ToPlayerData(),
		}
		payload, _ := proto.Marshal(dataRsp)
		rsp := &internalpb.Envelope{
			MsgId:     protocol.MsgLoadPlayerDataRsp,
			SessionId: env.SessionId,
			PlayerId:  env.PlayerId,
			Payload:   payload,
		}
		if err := transport.WriteEnvelope(conn, rsp); err != nil {
			log.Printf("[Game] send load data rsp failed: %v", err)
		}
	case protocol.MsgPlayerResumeReq:
		if _, err := s.players.ResumePlayer(context.Background(), env.SessionId, env.PlayerId); err != nil {
			log.Printf("[Game] resume player failed: %v", err)
		}
	case protocol.MsgPlayerOfflineNotify:
		s.players.MarkOffline(env.PlayerId)
	default:
		log.Printf("[Game] unknown msgID=%d", env.MsgId)
	}
}
