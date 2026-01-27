package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type RedisConfig struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
	UIDKey   string `json:"uid_key"`
}

type GateConfig struct {
	ListenAddr           string `json:"listen_addr"`
	ServiceAddr          string `json:"service_addr"`
	GameAddr             string `json:"game_addr"`
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
	ListenAddr string `json:"listen_addr"`
	ServerID   string `json:"server_id"`
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
