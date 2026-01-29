package redis_tools

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"

	"go.uber.org/zap"
)

type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

var redisClient *redis.Client
var redisCfg RedisConfig

func InitRedis(cfg RedisConfig) error {
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 200
	}
	if cfg.MinIdleConns <= 0 {
		cfg.MinIdleConns = 20
	}
	redisCfg = cfg

	redisClient = redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// 启动期健康检查
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	return redisClient.Ping(ctx).Err()
}

func RDB() *redis.Client {
	return redisClient
}

func StartHealthCheck(ctx context.Context, logger *zap.Logger, interval time.Duration) {
	if logger == nil {
		logger = zap.NewNop()
	}
	if interval <= 0 {
		interval = 10 * time.Second
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if redisClient == nil {
					continue
				}
				checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				err := redisClient.Ping(checkCtx).Err()
				cancel()
				if err == nil {
					continue
				}
				logger.Warn("redis ping failed",
					zap.String("reason", err.Error()),
				)
				_ = redisClient.Close()
				redisClient = redis.NewClient(&redis.Options{
					Addr:         redisCfg.Addr,
					Password:     redisCfg.Password,
					DB:           redisCfg.DB,
					PoolSize:     redisCfg.PoolSize,
					MinIdleConns: redisCfg.MinIdleConns,
					DialTimeout:  3 * time.Second,
					ReadTimeout:  3 * time.Second,
					WriteTimeout: 3 * time.Second,
				})
			}
		}
	}()
}
