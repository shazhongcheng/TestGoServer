package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type RedisConfig struct {
	Addr           string `json:"addr"`
	Password       string `json:"password"`
	DB             int    `json:"db"`
	UIDKey         string `json:"uid_key"`
	PoolSize       int    `json:"pool_size"`
	MinIdleConns   int    `json:"minIdle_conns"`
	HealthCheckSec int    `json:"health_check_sec"`
}

type GateConfig struct {
	ListenAddr           string `json:"listen_addr"`
	ServiceAddr          string `json:"service_addr"`
	GameAddr             string `json:"game_addr"`
	ServicePoolSize      int    `json:"service_pool_size"`
	HeartbeatIntervalSec int    `json:"heartbeat_interval_sec"`
	HeartbeatTimeoutSec  int    `json:"heartbeat_timeout_sec"`
	GCIntervalSec        int    `json:"gc_interval_sec"`
	LoginTimeoutSec      int    `json:"login_timeout_sec"`
	LoginRateLimitCount  int    `json:"login_rate_limit_count"`
	LoginRateLimitWindow int    `json:"login_rate_limit_window_sec"`
	UnknownMsgKickCount  int    `json:"unknown_msg_kick_count"`
	ConnReadTimeoutSec   int    `json:"conn_read_timeout_sec"`
	ConnWriteTimeoutSec  int    `json:"conn_write_timeout_sec"`
	ConnKeepAliveSec     int    `json:"conn_keepalive_sec"`
	MaxEnvelopeSize      uint32 `json:"max_envelope_size"`
}

type ServiceConfig struct {
	ListenAddr          string      `json:"listen_addr"`
	GameAddr            string      `json:"game_addr"`
	ConnReadTimeoutSec  int         `json:"conn_read_timeout_sec"`
	ConnWriteTimeoutSec int         `json:"conn_write_timeout_sec"`
	ConnKeepAliveSec    int         `json:"conn_keepalive_sec"`
	MaxEnvelopeSize     uint32      `json:"max_envelope_size"`
	Redis               RedisConfig `json:"redis"`
}

type GameConfig struct {
	ListenAddr          string      `json:"listen_addr"`
	ServerID            string      `json:"server_id"`
	ConnReadTimeoutSec  int         `json:"conn_read_timeout_sec"`
	ConnWriteTimeoutSec int         `json:"conn_write_timeout_sec"`
	ConnKeepAliveSec    int         `json:"conn_keepalive_sec"`
	MaxEnvelopeSize     uint32      `json:"max_envelope_size"`
	Redis               RedisConfig `json:"redis"`
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
