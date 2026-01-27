package redis_tools

import (
	"context"
	"fmt"
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

func (rd *RedisDao) Pipe() redis.Pipeliner {
	return rd.client.Pipeline()
}

/***************************** 针对redis操作自定义的方法 *****************************/

/*
	使用go-redis
	1. 方便使用，类型安全的API
	2. star多，用的人多，更新勤快（小时级别，redigo月级别）
	3. redis官方认证 https://redis.com/blog/go-redis-official-redis-client/
	4. go-redis特性更全，参考 https://redis.uptrace.dev/guide/go-redis-vs-redigo.html

	数据获取：
	方法一：
		val, err := redis.XXCommand().Result()
	方法二：
		res := redis.XXCommand()
		val := res.Val()
		err := res.Err()
	方法三：
		get := redis.Do()
		val, err := get.StringSlice() // 还包括其他方法，Bool、Int64Slice等（v8及以上版本）

	错误处理：
	go-redis get操作，res.Result() 返回数据和错误
	1. 数据不存在，err != nil && err == redis.Nil
	2. 其他错误，err != nil && err != redis.Nil
*/

/*
	哈希（Hash）操作
	string 类型的 field（字段） 和 value（值） 的映射表
	备注：
		key表示名为key的hash表，field表示元素名称，value表示field对应的值

	HSet(ctx, key, fieldAndValues)  		向名为key的hash中添加多个元素
	HGet(ctx, key, field)    				从名为key的hash中获取某个元素的值
	HMGet(ctx, key, fields)  				从名为key的hash中获取多个元素的值
	HGetAll(ctx, key)						从名为key的hash中获取所有元素
	HLen(ctx, key)							从名为key的hash中获取元素个数
	HExist(ctx, key, field)					从名为key的hash中检查某元素field是否存在
	HKeys(ctx, key)							从名为key的hash中获取所有元素名称
	HIncrByInt(ctx, key, field, int)		向名为key的hash中的元素field的值增加整数int
	HIncrByFloat(ctx, key, field, float)	向名为key的hash中的元素field的值增加浮点数int
	HVals(ctx, key)							从名为key的hash中获取所有元素的值
	HDel(ctx, key, fields)					从名为key的hash中删除多个元素
*/

// 往哈希表增加一个或多个元素field; 返回新增元素field的个数
// e.g. HSet(ctx, "hash1", "field1", 11, "field2", "22") -> 在hash1哈希表中添加field1和field2两个元素，值分别为11和"22"
func (rd *RedisDao) HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	val := rd.client.HSet(ctx, key, values)
	return val.Result()
}

func (rd *RedisDao) HMSet(ctx context.Context, key string, values ...interface{}) (bool, error) {
	val := rd.client.HMSet(ctx, key, values)
	return val.Result()
}

// 返回某个元素field对应的值，不存在key则返回redis.Nil
func (rd *RedisDao) HGet(ctx context.Context, key, filed string) (string, error) {
	val := rd.client.HGet(ctx, key, filed)
	return val.Result()
}

// 返回多个元素field对应的值slice，如果field不存在，则有返回相应数量的nil
// e.g. hash1只有元素field2 22，则 hmget hash1 field1 field2 field3 返回[<nil>, 22, <nil>]
func (rd *RedisDao) HMGet(ctx context.Context, key string, field ...string) ([]interface{}, error) {
	val := rd.client.HMGet(ctx, key, field...)
	return val.Result()
}

// 查不到返回空的map，err = nil
func (rd *RedisDao) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	val := rd.client.HGetAll(ctx, key)
	return val.Result()
}

// 返回元素的个数，若key不存在，返回0, err = nil
func (rd *RedisDao) HLen(ctx context.Context, key string) (int64, error) {
	val := rd.client.HLen(ctx, key)
	return val.Result()
}

// 是否存在某个元素field
func (rd *RedisDao) HExists(ctx context.Context, key, filed string) (bool, error) {
	val := rd.client.HExists(ctx, key, filed)
	return val.Result()
}

// 返回所有元素field，若key不存在返回空slice，err = nil
func (rd *RedisDao) HKeys(ctx context.Context, key string) ([]string, error) {
	val := rd.client.HKeys(ctx, key)
	return val.Result()
}

// 数字自增，分为整数自增和浮点数自增; 返回自增后的值；若key或者元素不存在则新增
func (rd *RedisDao) HIncrByInt(ctx context.Context, key, field string, increment int64) (int64, error) {
	val := rd.client.HIncrBy(ctx, key, field, increment)
	return val.Result()
}
func (rd *RedisDao) HIncrByFloat(ctx context.Context, key, field string, increment float64) (float64, error) {
	val := rd.client.HIncrByFloat(ctx, key, field, increment)
	return val.Result()
}

// 返回所有field对应的值slice，如果key不存在则返回空slice，err = nil
func (rd *RedisDao) HVals(ctx context.Context, key string) ([]string, error) {
	val := rd.client.HVals(ctx, key)
	return val.Result()
}

// 删除一个或多个元素field，返回删除成功的元素个数
func (rd *RedisDao) HDel(ctx context.Context, key string, fields ...string) (int64, error) {
	val := rd.client.HDel(ctx, key, fields...)
	return val.Result()
}

/*
	列表（List）操作
	String 元素类型的列表，按照插入顺序存储，可重复
	备注：
		key表示名为key的列表，value表示列表中的元素，索引从0开始，支持重复元素

	LPush(ctx, key, values)			在名为key的列表首部依次添加多个元素
	RPush(ctx, key, values)			在名为key的列表尾部依次添加多个元素
	LPop(ctx, key)					从名为key的列表首部移除并返回一个元素
	RPop(ctx, key)					从名为key的列表尾部移除并返回一个元素
	LRange(ctx, key, start, stop)	从名为key的列表中返回索引为[start,stop]范围内的元素
	LRem(ctx, key, count, value)	从名为key的列表中按照某种顺序删除count个和value相等的元素
	LLen(ctx, key)					返回名为key的列表元素的个数（列表长度）

	组合使用：
	队列（Lpush Rpop）、栈(Lpush Lpop)
*/

// 在列表首部添加一个或多个值，返回插入后列表的长度，若key不存在则创建新的列表
func (rd *RedisDao) LPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	val := rd.client.LPush(ctx, key, values)
	return val.Result()
}

// 在列表尾部添加一个或多个值，同LPush
func (rd *RedisDao) RPush(ctx context.Context, key string, values ...interface{}) (int64, error) {
	val := rd.client.RPush(ctx, key, values)
	return val.Result()
}

// 移出并返回列表的尾部元素，如果key不存在，err = redis.Nil
func (rd *RedisDao) LPop(ctx context.Context, key string) (string, error) {
	val := rd.client.LPop(ctx, key)
	return val.Result()
}

// 移出并返回列表的尾部元素，如果key不存在，err = redis.Nil
func (rd *RedisDao) RPop(ctx context.Context, key string) (string, error) {
	val := rd.client.RPop(ctx, key)
	return val.Result()
}

// 返回列表指定范围内的元素
func (rd *RedisDao) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	val := rd.client.LRange(ctx, key, start, stop)
	return val.Result()
}

// 从列表中移除元素，如果key不存在，返回空slice
// count > 0 : 从列表头部开始向尾部搜索，移除count个与value相等的元素
// count < 0 : 从列表尾部开始向头部搜索，移除count个与value相等的元素
// count = 0 : 移除列表所有与value相等的元素
func (rd *RedisDao) LRem(ctx context.Context, key string, count int64, value interface{}) error {
	return rd.client.LRem(ctx, key, count, value).Err()
}

// 返回列表元素的个数（列表长度），若key不存在，返回0
func (rd *RedisDao) LLen(ctx context.Context, key string) int64 {
	val := rd.client.LLen(ctx, key)
	return val.Val()
}

/*
	有序集合（Sorted Set）操作
	string 类型元素（member）的集合，每个元素关联一个双浮点型的数值（score）
	不支持重复元素，分数可以重复。元素按照分数从小到大排序，分数相同按照member字典序排序
	6.2.0 之后，废弃 zrevrange zrangebyscore, zrevrangebyscore, zrangebylex, zrevrangebylex，使用range代替
	备注：
		zrange bylex一般针对分数相同的元素
		key表示名为key的有序集合，member为元素，score为分数

	ZAdd(ctx, key, members)					从名为key的有序集合中添加多个元素
	ZIncrBy(ctx, key, member, increment)	从名为key的有序集合中对member的值增加increment
	ZRem(ctx, key, members)					从名为key的有序集合中删除多个元素
	ZRange(ctx, key, start, stop)			从名为key的有序集合中获取索引在
*/

// 在有序集合中添加元素，返回新增添加的元素个数，已有元素修改分数不算计数
func (rd *RedisDao) ZAdd(ctx context.Context, key string, members ...redis.Z) (int64, error) {
	val := rd.client.ZAdd(ctx, key, members...)
	return val.Result()
}

// 增加有序集合中元素的分数，不存在则自动创建，返回修改后的分数
func (rd *RedisDao) ZIncrBy(ctx context.Context, key, member string, increment float64) (float64, error) {
	val := rd.client.ZIncrBy(ctx, key, increment, member)
	return val.Result()
}

// 删除有序集合中的元素，返回实际删除的元素个数
func (rd *RedisDao) ZRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	val := rd.client.ZRem(ctx, key, members)
	return val.Result()
}

// 获取 [start, stop] 区间内的升序元素；不存在的元素会返回empty slice，区间默认为索引，bylex为字典序，byscore为分数
func (rd *RedisDao) ZRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	val := rd.client.ZRange(ctx, key, start, stop)
	return val.Result()
}

// 根据区间获取玩家列表
func (rd *RedisDao) ZRangeByScore(ctx context.Context, key string, opt *redis.ZRangeBy) ([]string, error) {
	val := rd.client.ZRangeByScore(ctx, key, opt)
	return val.Result()
}

func (rd *RedisDao) ZRangeByScoreWithScores(ctx context.Context, key string, opt *redis.ZRangeBy) ([]redis.Z, error) {
	val := rd.client.ZRangeByScoreWithScores(ctx, key, opt)
	return val.Result()
}

func (rd *RedisDao) ZRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	val := rd.client.ZRangeWithScores(ctx, key, start, stop)
	return val.Result()
}

// 获取 [start, stop] 区间内的降序元素；不存在的元素会返回empty slice，区间默认为索引
func (rd *RedisDao) ZRevRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	val := rd.client.ZRevRange(ctx, key, start, stop)
	return val.Result()
}

func (rd *RedisDao) ZRevRangeWithScores(ctx context.Context, key string, start, stop int64) ([]redis.Z, error) {
	val := rd.client.ZRevRangeWithScores(ctx, key, start, stop)
	return val.Result()
}

func (rd *RedisDao) ZRevRank(ctx context.Context, key string, member string) (int64, error) {
	val := rd.client.ZRevRank(ctx, key, member)
	return val.Result()
}

// 获取有序集合count个最小元素
func (rd *RedisDao) ZTopNMin(ctx context.Context, key string, count int64) ([]string, error) {
	if count > 0 {
		count--
	}
	return rd.ZRange(ctx, key, 0, count)
}

// 获取有序集合count个最大元素
func (rd *RedisDao) ZTopNMax(ctx context.Context, key string, count int64) ([]string, error) {
	if count > 0 {
		count--
	}
	return rd.ZRevRange(ctx, key, 0, count)
}

// 获取有序集合最大的分数，key不存在则返回err = redis.Nil
func (rd *RedisDao) ZMaxScore(ctx context.Context, key string) (float64, error) {
	var res float64
	val, err := rd.client.ZRevRangeWithScores(ctx, key, 0, 0).Result()
	if err != nil {
		return res, err
	}
	if len(val) == 0 {
		return res, redis.Nil
	}

	res = val[0].Score
	return res, err
}

// 获取有序集合元素的个数，不存在则返回0，err = nil
func (rd *RedisDao) ZCard(ctx context.Context, key string) (int64, error) {
	val := rd.client.ZCard(ctx, key)
	return val.Result()
}

// 获取某个分数区间元素的个数，不存在则返回0，err = nil
func (rd *RedisDao) ZCount(ctx context.Context, key string, start, stop float64) (int64, error) {
	val := rd.client.ZCount(ctx, key, fmt.Sprintf("%f", start), fmt.Sprintf("%f", stop))
	return val.Result()
}

// 获取某个元素的分数 不存在返回redis.Nil
func (rd *RedisDao) ZScore(ctx context.Context, key, member string) (float64, error) {
	val := rd.client.ZScore(ctx, key, member)
	return val.Result()
}

// 批量获取某个元素的分数
func (rd *RedisDao) ZMScore(ctx context.Context, key string, members ...string) ([]float64, error) {
	val := rd.client.ZMScore(ctx, key, members...)
	return val.Result()
}

func (rd *RedisDao) Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error) {
	val := rd.client.Eval(ctx, script, keys, args...)
	return val.Result()
}

func (rd *RedisDao) EvalSha(ctx context.Context, sha1 string, keys []string, args ...interface{}) (interface{}, error) {
	val := rd.client.EvalSha(ctx, sha1, keys, args...)
	return val.Result()
}

func (rd *RedisDao) ScriptLoad(ctx context.Context, script string) (string, error) {
	cmd := rd.client.ScriptLoad(ctx, script)
	return cmd.Result()
}

func (rd *RedisDao) ScriptFlush(ctx context.Context) (string, error) {
	cmd := rd.client.ScriptFlush(ctx)
	return cmd.Result()
}

/*
	无序集合（Set）操作
	String 类型的无序（member）集合，不支持重复元素
	可支持集合运算，交集、并集、差集
*/

// 向集合中添加元素，返回实际添加的元素个数
func (rd *RedisDao) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	val := rd.client.SAdd(ctx, key, members)
	return val.Result()
}

// 从集合中删除元素，返回被删除元素的个数，不存在返回0
func (rd *RedisDao) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	val := rd.client.SRem(ctx, key, members)
	return val.Result()
}

// 返回集合中的所有元素 (key不存在时，err = redis.Nil，data为空)
func (rd *RedisDao) SMembers(ctx context.Context, key string) ([]string, error) {
	val := rd.client.SMembers(ctx, key)
	return val.Result()
}

// 返回 members 是否在集合中 []bool
func (rd *RedisDao) SMIsMember(ctx context.Context, key string, members ...interface{}) ([]bool, error) {
	val := rd.client.SMIsMember(ctx, key, members)
	return val.Result()
}

// 返回集合元素的个数，若key不存在返回0，err = nil
func (rd *RedisDao) SCard(ctx context.Context, key string) (int64, error) {
	val := rd.client.SCard(ctx, key)
	return val.Result()
}

// 随机返回集合中N个元素，不删除，若key不存在返回空slice，err = nil
func (rd *RedisDao) SRandMemberN(ctx context.Context, key string, count int64) ([]string, error) {
	val := rd.client.SRandMemberN(ctx, key, count)
	return val.Result()
}

// 随机返回集合中count个元素并删除，若key不存在返回空slice，err = nil
func (rd *RedisDao) SRandPopN(ctx context.Context, key string, count int64) ([]string, error) {
	val := rd.client.SPopN(ctx, key, count)
	return val.Result()
}

/*
	键（Key）操作
*/

// 删除key，删除返回1，不存在返回0
func (rd *RedisDao) Del(ctx context.Context, keys ...string) (int64, error) {
	val := rd.client.Del(ctx, keys...)
	return val.Result()
}

/*
	字符串（String）操作
*/

// ex second px millisecond
func (rd *RedisDao) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return rd.client.Set(ctx, key, value, expiration).Err()
}

// 若key不存在，err = redis.Nil，data为空
func (rd *RedisDao) Get(ctx context.Context, key string) (string, error) {
	val := rd.client.Get(ctx, key)
	return val.Result()
}

func (rd *RedisDao) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	val := rd.client.MGet(ctx, keys...)
	return val.Result()
}

// GetInt 若key不存在，err = redis.Nil，data为空
func (rd *RedisDao) GetInt(ctx context.Context, key string) (int, error) {
	val := rd.client.Get(ctx, key)
	return val.Int()
}

// GetInt64 若key不存在，err = redis.Nil，data为空
func (rd *RedisDao) GetUInt64(ctx context.Context, key string) (uint64, error) {
	val := rd.client.Get(ctx, key)
	return val.Uint64()
}

// 只在键不存在时，才对键进行设置操作；成功返回true，失败返回false
func (rd *RedisDao) SetWhenNotExist(ctx context.Context, key string, value interface{}, expiration time.Duration) (
	bool, error,
) {
	val := rd.client.SetNX(ctx, key, value, expiration)
	return val.Result()
}

// 只在键已经存在时，才对键进行设置操作；成功返回true，失败返回false
func (rd *RedisDao) SetWhenExist(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return rd.client.SetXX(ctx, key, value, expiration).Err()
}

// Deprecated: replaced by SET with the EX argument. https://redis.io/commands/setex/
func (rd *RedisDao) SetEx(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return rd.client.SetEx(ctx, key, value, expiration).Err()
}

// expire
func (rd *RedisDao) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return rd.client.Expire(ctx, key, expiration).Err()
}

// 获取key的过期时间
// res == time.Duration(-2) key不存在
// res == time.Duration(-1) 没有设置过期时间
func (rd *RedisDao) TTL(ctx context.Context, key string) (time.Duration, error) {
	return rd.client.TTL(ctx, key).Result()
}

func (rd *RedisDao) Incr(ctx context.Context, key string) (int64, error) {
	val := rd.client.Incr(ctx, key)
	return val.Result()
}

func (rd *RedisDao) Watch(ctx context.Context, fn func(*redis.Tx) error, key string) error {
	return rd.client.Watch(ctx, fn, key)
}

// Bit 操作
func (rd *RedisDao) BitCount(ctx context.Context, key string) (int64, error) {
	return rd.client.BitCount(ctx, key, nil).Result()
}

func (rd *RedisDao) SetBit(ctx context.Context, key string, offset int64, value int) (int64, error) {
	return rd.client.SetBit(ctx, key, offset, value).Result()
}
