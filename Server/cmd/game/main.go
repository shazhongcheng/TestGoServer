package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"game-server/internal/config"
	"game-server/internal/game"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/game.yaml", "game config path")
	flag.Parse()

	var cfg config.GameConfig
	if err := config.Load(configPath, &cfg); err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server := game.NewServer(cfg.ListenAddr)
	log.Printf("[Game] listening on %s", cfg.ListenAddr)
	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatal(err)
	}
}
