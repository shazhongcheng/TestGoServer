// cmd/gate/main.go
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"game-server/internal/common/logging"
	"game-server/internal/config"
	"game-server/internal/gate"
	"game-server/internal/transport"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/gate.yaml", "gate config path")
	flag.Parse()

	logger, err := logging.NewLogger("gate")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	var cfg config.GateConfig
	if err := config.Load(configPath, &cfg); err != nil {
		logger.Error("load config failed",
			zap.String("reason", err.Error()),
			zap.Int("msg_id", 0),
			zap.Int64("session", 0),
			zap.Int64("player", 0),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		)
		os.Exit(1)
	}
	if cfg.MaxEnvelopeSize > 0 {
		transport.SetMaxEnvelopeSize(cfg.MaxEnvelopeSize)
	}

	// ========== 基础上下文 & 信号 ==========
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ========== Gate ==========
	g := gate.NewGate(logger)
	connOptions := transport.ConnOptions{
		ReadTimeout:  time.Duration(cfg.ConnReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.ConnWriteTimeoutSec) * time.Second,
		KeepAlive:    time.Duration(cfg.ConnKeepAliveSec) * time.Second,
	}
	g.UpdateConfig(
		time.Duration(cfg.HeartbeatIntervalSec)*time.Second,
		time.Duration(cfg.HeartbeatTimeoutSec)*time.Second,
		time.Duration(cfg.GCIntervalSec)*time.Second,
		time.Duration(cfg.LoginTimeoutSec)*time.Second,
		cfg.LoginRateLimitCount,
		time.Duration(cfg.LoginRateLimitWindow)*time.Second,
		cfg.UnknownMsgKickCount,
		connOptions,
	)
	g.Start(ctx)
	g.ConnectService(ctx, cfg.ServiceAddr, cfg.ServicePoolSize)

	enableTCP := cfg.EnableTCP
	enableWS := cfg.EnableWebSocket
	if !enableTCP && !enableWS {
		enableTCP = true
	}

	var tcpListener net.Listener
	if enableTCP {
		addr := cfg.ListenAddr
		ln, err := net.Listen("tcp", addr)
		if err != nil {
			logger.Error("gate listen failed",
				zap.String("reason", err.Error()),
				zap.Int("msg_id", 0),
				zap.Int64("session", 0),
				zap.Int64("player", 0),
				zap.Int64("conn_id", 0),
				zap.String("trace_id", ""),
			)
			os.Exit(1)
		}
		tcpListener = ln
		logger.Info("gate listening (tcp)",
			zap.Int("msg_id", 0),
			zap.Int64("session", 0),
			zap.Int64("player", 0),
			zap.String("reason", ""),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
			zap.String("addr", addr),
		)

		go func() {
			for {
				conn, err := ln.Accept()
				if err != nil {
					select {
					case <-ctx.Done():
						return
					default:
						logger.Warn("accept error",
							zap.Int("msg_id", 0),
							zap.Int64("session", 0),
							zap.Int64("player", 0),
							zap.String("reason", err.Error()),
							zap.Int64("conn_id", 0),
							zap.String("trace_id", ""),
						)
						continue
					}
				}

				go handleConn(g, conn)
			}
		}()
	}

	var wsServer *http.Server
	if enableWS {
		wsAddr := cfg.WebSocketListenAddr
		if wsAddr == "" {
			logger.Error("websocket listen address required",
				zap.String("reason", "missing websocket_listen_addr"),
				zap.Int("msg_id", 0),
				zap.Int64("session", 0),
				zap.Int64("player", 0),
				zap.Int64("conn_id", 0),
				zap.String("trace_id", ""),
			)
			os.Exit(1)
		}
		wsPath := cfg.WebSocketPath
		if wsPath == "" {
			wsPath = "/ws"
		}
		upgrader := websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		mux := http.NewServeMux()
		mux.HandleFunc(wsPath, func(w http.ResponseWriter, r *http.Request) {
			wsConn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				logger.Warn("websocket upgrade failed",
					zap.String("reason", err.Error()),
					zap.Int("msg_id", 0),
					zap.Int64("session", 0),
					zap.Int64("player", 0),
					zap.Int64("conn_id", 0),
					zap.String("trace_id", ""),
				)
				return
			}
			go handleWSConn(g, wsConn, cfg.WebSocketUseJSON)
		})
		wsServer = &http.Server{
			Addr:    wsAddr,
			Handler: mux,
		}
		go func() {
			if err := wsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("websocket listen failed",
					zap.String("reason", err.Error()),
					zap.Int("msg_id", 0),
					zap.Int64("session", 0),
					zap.Int64("player", 0),
					zap.Int64("conn_id", 0),
					zap.String("trace_id", ""),
				)
				cancel()
			}
		}()
		logger.Info("gate listening (websocket)",
			zap.Int("msg_id", 0),
			zap.Int64("session", 0),
			zap.Int64("player", 0),
			zap.String("reason", ""),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
			zap.String("addr", wsAddr),
			zap.String("path", wsPath),
		)
	}

	// ========== 等待退出 ==========
	<-sigCh
	logger.Info("gate shutting down",
		zap.Int("msg_id", 0),
		zap.Int64("session", 0),
		zap.Int64("player", 0),
		zap.String("reason", ""),
		zap.Int64("conn_id", 0),
		zap.String("trace_id", ""),
	)

	cancel()
	if tcpListener != nil {
		_ = tcpListener.Close()
	}
	if wsServer != nil {
		_ = wsServer.Shutdown(context.Background())
	}

	time.Sleep(500 * time.Millisecond)
	logger.Info("gate exited",
		zap.Int("msg_id", 0),
		zap.Int64("session", 0),
		zap.Int64("player", 0),
		zap.String("reason", ""),
		zap.Int64("conn_id", 0),
		zap.String("trace_id", ""),
	)
}

func handleConn(g *gate.Gate, netConn net.Conn) {
	c := gate.NewConn(netConn, g)

	// 创建 Session
	//g.NewSession(c)

	g.Logger().Info("gate new connection",
		zap.Int("msg_id", 0),
		zap.Int64("player", 0),
		zap.String("reason", ""),
		zap.Int64("sesson_Id", c.SessonId()),
		zap.String("trace_id", c.TraceID()),
	)

	// 启动读循环
	c.ReadLoop()
}

func handleWSConn(g *gate.Gate, wsConn *websocket.Conn, useJSON bool) {
	c := gate.NewWSConn(wsConn, g, useJSON)

	g.Logger().Info("gate new websocket connection",
		zap.Int("msg_id", 0),
		zap.Int64("player", 0),
		zap.String("reason", ""),
		zap.Int64("sesson_Id", c.SessonId()),
		zap.String("trace_id", c.TraceID()),
	)

	c.ReadLoop()
}
