// cmd/service/main.go
package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"game-server/internal/config"
	"game-server/internal/db"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/service"
	"game-server/internal/service/modules/chat"
	"game-server/internal/service/modules/login"
)

func main() {
	srv := service.NewServer()

	var configPath string
	flag.StringVar(&configPath, "config", "configs/service.yaml", "service config path")
	flag.Parse()

	var cfg config.ServiceConfig
	if err := config.Load(configPath, &cfg); err != nil {
		log.Fatal(err)
	}

	uidClient := db.NewRedisClient(cfg.Redis)
	loginSvc := login.NewLoginService(uidClient)

	if err := srv.RegisterModule(login.NewModule(loginSvc)); err != nil {
		log.Fatal(err)
	}
	if err := srv.RegisterModule(&chat.Module{}); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	gameRouter := service.NewGameRouter(cfg.GameAddr)
	netServer := service.NewNetServer(srv, gameRouter)
	gameRouter.Start(ctx, func(env *internalpb.Envelope) {
		if err := netServer.ForwardToGate(env); err != nil {
			log.Printf("[Service] forward game env failed: %v", err)
		}
	})

	log.Printf("[Service] listening on %s", cfg.ListenAddr)
	if err := netServer.ListenAndServe(ctx, cfg.ListenAddr); err != nil {
		log.Fatal(err)
	}
}
