// cmd/gate/main.go
package main

import (
	"context"
	"game-server/internal/common/selflog"
	"log"

	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"game-server/internal/gate"
	"game-server/internal/service"
)

func main() {
	// ========== 基础上下文 & 信号 ==========
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// ========== Service（先用本地内存版） ==========
	svc := service.NewServer()

	// TODO：后面你会在这里 RegisterModule(login/chat/...)
	// svc.RegisterModule(...)

	// ========== Gate ==========
	logger := selflog.NewStdLogger("Gate")
	g := gate.NewGate(logger, svc)
	g.Start(ctx)

	// ========== TCP Listener ==========
	addr := ":9000"
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
	g.NewSession(c)

	log.Println("[Gate] new connection")

	// 启动读循环
	c.ReadLoop()
}
