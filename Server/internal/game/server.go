package game

import (
	"context"
	"net"

	"game-server/internal/player"
	"game-server/internal/protocol"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/transport"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type Server struct {
	addr    string
	players *PlayerManager
	logger  *zap.Logger
}

func NewServer(addr string, store player.Store, logger *zap.Logger) *Server {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Server{
		addr:    addr,
		players: NewPlayerManager(store),
		logger:  logger,
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
	buffered := transport.NewBufferedConn(conn)
	defer buffered.Close()
	for {
		env, err := buffered.ReadEnvelope()
		if err != nil {
			return
		}
		go s.handleEnvelope(buffered, env)
	}
}

func (s *Server) handleEnvelope(conn *transport.BufferedConn, env *internalpb.Envelope) {
	s.logger.Info("game envelope received",
		zap.Int("msg_id", int(env.MsgId)),
		zap.Int64("session", env.SessionId),
		zap.Int64("player", env.PlayerId),
		zap.String("reason", ""),
		zap.Int64("conn_id", 0),
		zap.String("trace_id", ""),
	)
	switch int(env.MsgId) {
	case protocol.MsgPlayerEnterGameReq:
		s.logger.Info("player enter",
			zap.Int("msg_id", int(env.MsgId)),
			zap.Int64("session", env.SessionId),
			zap.Int64("player", env.PlayerId),
			zap.String("reason", "player_enter"),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		)
		info, err := s.players.EnsurePlayer(context.Background(), env.SessionId, env.PlayerId)
		if err != nil {
			s.logger.Warn("load player failed",
				zap.Int("msg_id", int(env.MsgId)),
				zap.Int64("session", env.SessionId),
				zap.Int64("player", env.PlayerId),
				zap.String("reason", err.Error()),
				zap.Int64("conn_id", 0),
				zap.String("trace_id", ""),
			)
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
		if err := conn.WriteEnvelope(rsp); err != nil {
			s.logger.Warn("send enter rsp failed",
				zap.Int("msg_id", int(env.MsgId)),
				zap.Int64("session", env.SessionId),
				zap.Int64("player", env.PlayerId),
				zap.String("reason", err.Error()),
				zap.Int64("conn_id", 0),
				zap.String("trace_id", ""),
			)
		}
	case protocol.MsgLoadPlayerDataReq:
		info, err := s.players.EnsurePlayer(context.Background(), env.SessionId, env.PlayerId)
		s.logger.Debug("send MsgLoadPlayerDataReq",
			zap.Int("msg_id", int(env.MsgId)),
			zap.Int64("session", env.SessionId),
			zap.Int64("player", env.PlayerId),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		)
		if err != nil {
			s.logger.Warn("load player data failed",
				zap.Int("msg_id", int(env.MsgId)),
				zap.Int64("session", env.SessionId),
				zap.Int64("player", env.PlayerId),
				zap.String("reason", err.Error()),
				zap.Int64("conn_id", 0),
				zap.String("trace_id", ""),
			)
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
		if err := conn.WriteEnvelope(rsp); err != nil {
			s.logger.Warn("send load data rsp failed",
				zap.Int("msg_id", int(env.MsgId)),
				zap.Int64("session", env.SessionId),
				zap.Int64("player", env.PlayerId),
				zap.String("reason", err.Error()),
				zap.Int64("conn_id", 0),
				zap.String("trace_id", ""),
			)
		}
	case protocol.MsgPlayerResumeReq:
		if _, err := s.players.ResumePlayer(context.Background(), env.SessionId, env.PlayerId); err != nil {
			s.logger.Warn("resume player failed",
				zap.Int("msg_id", int(env.MsgId)),
				zap.Int64("session", env.SessionId),
				zap.Int64("player", env.PlayerId),
				zap.String("reason", err.Error()),
				zap.Int64("conn_id", 0),
				zap.String("trace_id", ""),
			)
		}
	case protocol.MsgPlayerOfflineNotify:
		s.players.MarkOffline(env.PlayerId)
	default:
		s.logger.Warn("unknown msgID",
			zap.Int("msg_id", int(env.MsgId)),
			zap.Int64("session", env.SessionId),
			zap.Int64("player", env.PlayerId),
			zap.String("reason", "unknown_msg_id"),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		)
	}
}
