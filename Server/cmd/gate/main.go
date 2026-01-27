// cmd/gate/main.go
package main

import (
	"context"
	"flag"
	"game-server/internal/common/selflog"
	"game-server/internal/config"
	"log"

	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"game-server/internal/gate"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/gate.yaml", "gate config path")
	flag.Parse()

	var cfg config.GateConfig
	if err := config.Load(configPath, &cfg); err != nil {
		log.Fatal(err)
	}

	// ========== 基础上下文 & 信号 ==========
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ========== Gate ==========
	logger := selflog.NewStdLogger("Gate")
	g := gate.NewGate(logger)
	g.UpdateConfig(
		time.Duration(cfg.HeartbeatIntervalSec)*time.Second,
		time.Duration(cfg.HeartbeatTimeoutSec)*time.Second,
		time.Duration(cfg.GCIntervalSec)*time.Second,
	)
	g.Start(ctx)
	g.ConnectService(ctx, cfg.ServiceAddr)

	// ========== TCP Listener ==========
	addr := cfg.ListenAddr
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("gate listen failed: %v", err)
	}
	log.Println("[Gate] listening on", addr)

	// ========== Accept Loop ==========
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					log.Println("accept error:", err)
					continue
				}
			}

			go handleConn(g, conn)
		}
	}()

	// ========== 等待退出 ==========
	<-sigCh
	log.Println("[Gate] shutting down...")

	cancel()
	_ = ln.Close()

	time.Sleep(500 * time.Millisecond)
	log.Println("[Gate] exited")
}

func handleConn(g *gate.Gate, netConn net.Conn) {
	defer netConn.Close()

	c := gate.NewConn(netConn, g)

	// 创建 Session
	//g.NewSession(c)

	log.Println("[Gate] new connection")

	// 启动读循环
	c.ReadLoop()
}
