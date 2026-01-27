// cmd/service/main.go
package main

import (
	"context"
	"flag"
	"game-server/internal/db/redis_tools"
	"log"
	"os"
	"os/signal"
	"syscall"

	"game-server/internal/common/logging"
	"game-server/internal/config"
	"game-server/internal/player"
	"game-server/internal/protocol/internalpb"
	"game-server/internal/service"
	"game-server/internal/service/modules/chat"
	"game-server/internal/service/modules/login"
	"go.uber.org/zap"
)

func main() {
	logger, err := logging.NewLogger("service")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	var configPath string
	flag.StringVar(&configPath, "config", "configs/service.yaml", "service config path")
	flag.Parse()

	var cfg config.ServiceConfig
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

	srv := service.NewServer(logger)

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

	playerStore := player.NewRedisStore(redis_tools.NewRedisDao())
	loginSvc := login.NewLoginService(redis_tools.NewRedisDao(), playerStore)

	if err := srv.RegisterModule(login.NewModule(loginSvc)); err != nil {
		logger.Error("register login module failed",
			zap.String("reason", err.Error()),
			zap.Int("msg_id", 0),
			zap.Int64("session", 0),
			zap.Int64("player", 0),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		)
		os.Exit(1)
	}
	if err := srv.RegisterModule(&chat.Module{}); err != nil {
		logger.Error("register chat module failed",
			zap.String("reason", err.Error()),
			zap.Int("msg_id", 0),
			zap.Int64("session", 0),
			zap.Int64("player", 0),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	gameRouter := service.NewGameRouter(cfg.GameAddr)
	netServer := service.NewNetServer(srv, gameRouter)
	gameRouter.Start(ctx, func(env *internalpb.Envelope) {
		if err := netServer.ForwardToGate(env); err != nil {
			logger.Warn("forward game env failed",
				zap.Int("msg_id", int(env.MsgId)),
				zap.Int64("session", env.SessionId),
				zap.Int64("player", env.PlayerId),
				zap.String("reason", err.Error()),
				zap.Int64("conn_id", 0),
				zap.String("trace_id", ""),
			)
		}
	})

	logger.Info("service listening",
		zap.Int("msg_id", 0),
		zap.Int64("session", 0),
		zap.Int64("player", 0),
		zap.String("reason", ""),
		zap.Int64("conn_id", 0),
		zap.String("trace_id", ""),
		zap.String("addr", cfg.ListenAddr),
	)
	if err := netServer.ListenAndServe(ctx, cfg.ListenAddr); err != nil {
		logger.Error("service listen failed",
			zap.String("reason", err.Error()),
			zap.Int("msg_id", 0),
			zap.Int64("session", 0),
			zap.Int64("player", 0),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ""),
		)
		os.Exit(1)
	}
}
