package redis_tools

import (
	"context"
	"github.com/redis/go-redis/v9"
	"time"
)

type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
}

var redisClient *redis.Client

func InitRedis(cfg RedisConfig) error {
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 200
	}
	if cfg.MinIdleConns <= 0 {
		cfg.MinIdleConns = 20
	}

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
