package player_db

import (
	"context"
	"encoding/json"
	"fmt"
	"game-server/internal/db/redis_tools"
	"github.com/redis/go-redis/v9"
	"strconv"
)

type Store interface {
	LoadRoleID(ctx context.Context, accountID string) (int64, bool, error)
	SaveRoleID(ctx context.Context, accountID string, roleID int64) error
	LoadProfile(ctx context.Context, roleID int64) (*PlayerProfile, bool, error)
	SaveProfile(ctx context.Context, profile *PlayerProfile) error
}

type RedisStore struct {
	dao *redis_tools.RedisDao
}

func NewRedisStore(dao *redis_tools.RedisDao) *RedisStore {
	return &RedisStore{dao: dao}
}

// =======================
// Account ↔ Role
// =======================
func (s *RedisStore) LoadRoleID(
	ctx context.Context,
	accountID string,
) (int64, bool, error) {

	if accountID == "" {
		return 0, false, nil
	}

	key := redis_tools.AccountRoleKey(accountID)
	val, err := s.dao.GetString(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return 0, false, nil
		}
		return 0, false, err
	}

	roleID, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse role id: %w", err)
	}

	return roleID, true, nil
}

func (s *RedisStore) SaveRoleID(
	ctx context.Context,
	accountID string,
	roleID int64,
) error {

	if accountID == "" || roleID == 0 {
		return nil
	}

	key := redis_tools.AccountRoleKey(accountID)
	return s.dao.SetWithTTL(ctx, key, roleID, 0)
}

// =======================
// Player Profile
// =======================
func (s *RedisStore) LoadProfile(
	ctx context.Context,
	roleID int64,
) (*PlayerProfile, bool, error) {

	if roleID == 0 {
		return nil, false, nil
	}

	key := redis_tools.PlayerProfileKey(roleID)
	val, err := s.dao.GetString(ctx, key)
	if err != nil {
		if err == redis.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var profile PlayerProfile
	if err := json.Unmarshal([]byte(val), &profile); err != nil {
		return nil, false, fmt.Errorf("decode profile: %w", err)
	}

	return &profile, true, nil
}

func (s *RedisStore) SaveProfile(
	ctx context.Context,
	profile *PlayerProfile,
) error {

	if profile == nil || profile.RoleID == 0 {
		return nil
	}

	data, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("encode profile: %w", err)
	}

	key := redis_tools.PlayerProfileKey(profile.RoleID)
	return s.dao.SetWithTTL(ctx, key, data, 0)
}
