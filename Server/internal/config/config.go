package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type RedisConfig struct {
	Addr         string `json:"addr"`
	Password     string `json:"password"`
	DB           int    `json:"db"`
	UIDKey       string `json:"uid_key"`
	PoolSize     int    `json:"pool_size"`
	MinIdleConns int    `json:"minIdle_conns"`
}

type GateConfig struct {
	ListenAddr           string `json:"listen_addr"`
	ServiceAddr          string `json:"service_addr"`
	GameAddr             string `json:"game_addr"`
	ServicePoolSize      int    `json:"service_pool_size"`
	HeartbeatIntervalSec int    `json:"heartbeat_interval_sec"`
	HeartbeatTimeoutSec  int    `json:"heartbeat_timeout_sec"`
	GCIntervalSec        int    `json:"gc_interval_sec"`
}

type ServiceConfig struct {
	ListenAddr string      `json:"listen_addr"`
	GameAddr   string      `json:"game_addr"`
	Redis      RedisConfig `json:"redis"`
}

type GameConfig struct {
	ListenAddr string      `json:"listen_addr"`
	ServerID   string      `json:"server_id"`
	Redis      RedisConfig `json:"redis"`
}

func Load(path string, out any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read config %s: %w", path, err)
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	return nil
}
