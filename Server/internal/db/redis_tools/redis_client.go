package redis_tools

import (
	"context"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"sync"
	"time"
)

type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

var (
	mu          sync.RWMutex
	redisClient *redis.Client
	redisCfg    RedisConfig
)

// InitRedis：只做一次初始化
func InitRedis(cfg RedisConfig) error {
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 200
	}
	if cfg.MinIdleConns <= 0 {
		cfg.MinIdleConns = 20
	}

	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Addr,
		Password:     cfg.Password,
		DB:           cfg.DB,
		PoolSize:     cfg.PoolSize,
		MinIdleConns: cfg.MinIdleConns,
		DialTimeout:  3 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return err
	}

	mu.Lock()
	redisCfg = cfg
	redisClient = client
	mu.Unlock()

	return nil
}

// RDB：统一出口（永不返回 nil）
func RDB() *redis.Client {
	mu.RLock()
	c := redisClient
	mu.RUnlock()
	return c
}

// StartHealthCheck
// ⚠️ 只做“状态探测”，绝不 Close / 重建 client
func StartHealthCheck(
	ctx context.Context,
	logger *zap.Logger,
	interval time.Duration,
) {
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
				client := RDB()
				if client == nil {
					logger.Warn("redis client not initialized")
					continue
				}

				checkCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
				err := client.Ping(checkCtx).Err()
				cancel()

				if err != nil {
					logger.Warn("redis ping failed",
						zap.String("addr", redisCfg.Addr),
						zap.String("reason", err.Error()),
					)
				}
			}
		}
	}()
}
