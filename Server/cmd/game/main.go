package main

import (
	"context"
	"flag"
	"log"
	"os/signal"
	"syscall"

	"game-server/internal/config"
	"game-server/internal/db"
	"game-server/internal/game"
	"game-server/internal/player"
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

	redisClient := db.NewRedisClient(cfg.Redis)
	playerStore := player.NewRedisStore(redisClient)
	server := game.NewServer(cfg.ListenAddr, playerStore)
	log.Printf("[Game] listening on %s", cfg.ListenAddr)
	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatal(err)
	}
}
