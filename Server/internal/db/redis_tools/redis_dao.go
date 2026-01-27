package redis_tools

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisDao struct {
	client *redis.Client
}

func NewRedisDao() *RedisDao {
	return &RedisDao{
		client: RDB(),
	}
}

//
// =======================
// UID / Login 相关
// =======================
//

// 全局 UID（Login 用）
func (rd *RedisDao) NextUID(ctx context.Context) (int64, error) {
	return rd.client.Incr(ctx, KeyUIDNext).Result()
}

//
// =======================
// Player Base
// =======================
//

func (rd *RedisDao) LoadPlayerBase(
	ctx context.Context,
	playerID int64,
) (map[string]string, error) {

	key := KeyPlayerBase(playerID)
	return rd.client.HGetAll(ctx, key).Result()
}

func (rd *RedisDao) SavePlayerBase(
	ctx context.Context,
	playerID int64,
	fields map[string]interface{},
) error {

	key := KeyPlayerBase(playerID)
	return rd.client.HSet(ctx, key, fields).Err()
}

//
// =======================
// 排行榜（ZSet）
// =======================
//

func (rd *RedisDao) UpdateRankScore(
	ctx context.Context,
	rank string,
	playerID int64,
	score float64,
) error {

	key := KeyRank(rank)
	return rd.client.ZAdd(ctx, key, redis.Z{
		Member: playerID,
		Score:  score,
	}).Err()
}

func (rd *RedisDao) GetTopN(
	ctx context.Context,
	rank string,
	n int64,
) ([]redis.Z, error) {

	key := KeyRank(rank)
	return rd.client.ZRevRangeWithScores(ctx, key, 0, n-1).Result()
}

//
// =======================
// 通用工具
// =======================
//

func (rd *RedisDao) SetWithTTL(
	ctx context.Context,
	key string,
	value interface{},
	ttl time.Duration,
) error {

	return rd.client.Set(ctx, key, value, ttl).Err()
}

func (rd *RedisDao) GetString(
	ctx context.Context,
	key string,
) (string, error) {

	return rd.client.Get(ctx, key).Result()
}

func (rd *RedisDao) Exists(
	ctx context.Context,
	key string,
) (bool, error) {

	n, err := rd.client.Exists(ctx, key).Result()
	return n == 1, err
}
