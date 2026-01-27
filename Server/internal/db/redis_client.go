package db

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"game-server/internal/config"
)

type RedisClient struct {
	addr     string
	password string
	db       int
	uidKey   string
	timeout  time.Duration
}

func NewRedisClient(cfg config.RedisConfig) *RedisClient {
	return &RedisClient{
		addr:     cfg.Addr,
		password: cfg.Password,
		db:       cfg.DB,
		uidKey:   cfg.UIDKey,
		timeout:  3 * time.Second,
	}
}

func (c *RedisClient) NextUID(ctx context.Context) (int64, error) {
	conn, err := net.DialTimeout("tcp", c.addr, c.timeout)
	if err != nil {
		return 0, fmt.Errorf("redis dial: %w", err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(c.timeout))
	reader := bufio.NewReader(conn)

	if c.password != "" {
		if err := writeCommand(conn, "AUTH", c.password); err != nil {
			return 0, err
		}
		if _, err := readSimpleReply(reader); err != nil {
			return 0, err
		}
	}

	if c.db != 0 {
		if err := writeCommand(conn, "SELECT", strconv.Itoa(c.db)); err != nil {
			return 0, err
		}
		if _, err := readSimpleReply(reader); err != nil {
			return 0, err
		}
	}

	key := c.uidKey
	if key == "" {
		key = "uid:next"
	}
	if err := writeCommand(conn, "INCR", key); err != nil {
		return 0, err
	}
	return readIntegerReply(reader)
}

func writeCommand(conn net.Conn, parts ...string) error {
	if _, err := fmt.Fprintf(conn, "*%d\r\n", len(parts)); err != nil {
		return err
	}
	for _, part := range parts {
		if _, err := fmt.Fprintf(conn, "$%d\r\n%s\r\n", len(part), part); err != nil {
			return err
		}
	}
	return nil
}

func readSimpleReply(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	if len(line) == 0 {
		return "", fmt.Errorf("empty redis reply")
	}
	switch line[0] {
	case '+':
		return strings.TrimSpace(line[1:]), nil
	case '-':
		return "", fmt.Errorf("redis error: %s", strings.TrimSpace(line[1:]))
	default:
		return "", fmt.Errorf("unexpected redis reply: %q", line)
	}
}

func readIntegerReply(reader *bufio.Reader) (int64, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, err
	}
	if len(line) == 0 {
		return 0, fmt.Errorf("empty redis reply")
	}
	if line[0] == '-' {
		return 0, fmt.Errorf("redis error: %s", strings.TrimSpace(line[1:]))
	}
	if line[0] != ':' {
		return 0, fmt.Errorf("unexpected redis reply: %q", line)
	}
	value := strings.TrimSpace(line[1:])
	return strconv.ParseInt(value, 10, 64)
}
