package player

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"game-server/internal/db"
)

type Store interface {
	LoadRoleID(ctx context.Context, accountID string) (int64, bool, error)
	SaveRoleID(ctx context.Context, accountID string, roleID int64) error
	LoadProfile(ctx context.Context, roleID int64) (*PlayerProfile, bool, error)
	SaveProfile(ctx context.Context, profile *PlayerProfile) error
}

type RedisStore struct {
	client *db.RedisClient
}

func NewRedisStore(client *db.RedisClient) *RedisStore {
	return &RedisStore{client: client}
}

func (s *RedisStore) LoadRoleID(ctx context.Context, accountID string) (int64, bool, error) {
	if accountID == "" {
		return 0, false, nil
	}
	value, ok, err := s.client.GetString(ctx, AccountRoleKey(accountID))
	if err != nil || !ok {
		return 0, ok, err
	}
	roleID, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse role id: %w", err)
	}
	return roleID, true, nil
}

func (s *RedisStore) SaveRoleID(ctx context.Context, accountID string, roleID int64) error {
	if accountID == "" {
		return nil
	}
	return s.client.SetString(ctx, AccountRoleKey(accountID), strconv.FormatInt(roleID, 10))
}

func (s *RedisStore) LoadProfile(ctx context.Context, roleID int64) (*PlayerProfile, bool, error) {
	if roleID == 0 {
		return nil, false, nil
	}
	value, ok, err := s.client.GetString(ctx, PlayerProfileKey(roleID))
	if err != nil || !ok {
		return nil, ok, err
	}
	var profile PlayerProfile
	if err := json.Unmarshal([]byte(value), &profile); err != nil {
		return nil, false, fmt.Errorf("decode profile: %w", err)
	}
	return &profile, true, nil
}

func (s *RedisStore) SaveProfile(ctx context.Context, profile *PlayerProfile) error {
	if profile == nil {
		return nil
	}
	data, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("encode profile: %w", err)
	}
	return s.client.SetString(ctx, PlayerProfileKey(profile.RoleID), string(data))
}
