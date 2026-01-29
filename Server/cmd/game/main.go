package main

import (
	"context"
	"flag"
	"game-server/internal/db/redis_tools"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"game-server/internal/common/logging"
	"game-server/internal/config"
	"game-server/internal/game"
	"game-server/internal/player_db"
	"game-server/internal/transport"
	"go.uber.org/zap"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "configs/game.yaml", "game config path")
	flag.Parse()

	logger, err := logging.NewLogger("game")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	var cfg config.GameConfig
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
	connOptions := transport.ConnOptions{
		ReadTimeout:  time.Duration(cfg.ConnReadTimeoutSec) * time.Second,
		WriteTimeout: time.Duration(cfg.ConnWriteTimeoutSec) * time.Second,
		KeepAlive:    time.Duration(cfg.ConnKeepAliveSec) * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// 2️⃣ 初始化 Redis（关键）
	if err := redis_tools.InitRedis(redis_tools.RedisConfig{
		Addr:         cfg.Redis.Addr,
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		PoolSize:     cfg.Redis.PoolSize,
		MinIdleConns: cfg.Redis.MinIdleConns,
	}); err != nil {
		log.Fatalf("init redis failed: %v", err)
	}
	redis_tools.StartHealthCheck(ctx, logger, time.Duration(cfg.Redis.HealthCheckSec)*time.Second)

	playerStore := player_db.NewRedisStore(redis_tools.NewRedisDao())
	server := game.NewServer(cfg.ListenAddr, playerStore, logger, connOptions, 30*time.Second)
	logger.Info("game listening",
		zap.Int("msg_id", 0),
		zap.Int64("session", 0),
		zap.Int64("player", 0),
		zap.String("reason", ""),
		zap.Int64("conn_id", 0),
		zap.String("trace_id", ""),
		zap.String("addr", cfg.ListenAddr),
	)
	if err := server.ListenAndServe(ctx); err != nil {
		log.Fatal(err)
	}
}
