// cmd/gate/main.go
package main

import (
	"context"
	"flag"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"game-server/internal/common/logging"
	"game-server/internal/config"
	"game-server/internal/gate"
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

	// ========== 基础上下文 & 信号 ==========
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ========== Gate ==========
	g := gate.NewGate(logger)
	g.UpdateConfig(
		time.Duration(cfg.HeartbeatIntervalSec)*time.Second,
		time.Duration(cfg.HeartbeatTimeoutSec)*time.Second,
		time.Duration(cfg.GCIntervalSec)*time.Second,
	)
	g.Start(ctx)
	g.ConnectService(ctx, cfg.ServiceAddr, cfg.ServicePoolSize)

	// ========== TCP Listener ==========
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
	logger.Info("gate listening",
		zap.Int("msg_id", 0),
		zap.Int64("session", 0),
		zap.Int64("player", 0),
		zap.String("reason", ""),
		zap.Int64("conn_id", 0),
		zap.String("trace_id", ""),
		zap.String("addr", addr),
	)

	// ========== Accept Loop ==========
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
	_ = ln.Close()

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
