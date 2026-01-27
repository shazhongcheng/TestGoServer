package db

import (
	"bufio"
	"context"
	"fmt"
	"io"
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
	key := c.uidKey
	if key == "" {
		key = "uid:next"
	}
	conn, reader, err := c.dial()
	if err != nil {
		return 0, err
	}
	defer conn.Close()

	if err := writeCommand(conn, "INCR", key); err != nil {
		return 0, err
	}
	return readIntegerReply(reader)
}

func (c *RedisClient) GetString(ctx context.Context, key string) (string, bool, error) {
	conn, reader, err := c.dial()
	if err != nil {
		return "", false, err
	}
	defer conn.Close()

	if err := writeCommand(conn, "GET", key); err != nil {
		return "", false, err
	}
	return readBulkStringReply(reader)
}

func (c *RedisClient) SetString(ctx context.Context, key, value string) error {
	conn, reader, err := c.dial()
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := writeCommand(conn, "SET", key, value); err != nil {
		return err
	}
	_, err = readSimpleReply(reader)
	return err
}

func (c *RedisClient) dial() (net.Conn, *bufio.Reader, error) {
	conn, err := net.DialTimeout("tcp", c.addr, c.timeout)
	if err != nil {
		return nil, nil, fmt.Errorf("redis dial: %w", err)
	}

	_ = conn.SetDeadline(time.Now().Add(c.timeout))
	reader := bufio.NewReader(conn)

	if c.password != "" {
		if err := writeCommand(conn, "AUTH", c.password); err != nil {
			_ = conn.Close()
			return nil, nil, err
		}
		if _, err := readSimpleReply(reader); err != nil {
			_ = conn.Close()
			return nil, nil, err
		}
	}

	if c.db != 0 {
		if err := writeCommand(conn, "SELECT", strconv.Itoa(c.db)); err != nil {
			_ = conn.Close()
			return nil, nil, err
		}
		if _, err := readSimpleReply(reader); err != nil {
			_ = conn.Close()
			return nil, nil, err
		}
	}

	return conn, reader, nil
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

func readBulkStringReply(reader *bufio.Reader) (string, bool, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", false, err
	}
	if len(line) == 0 {
		return "", false, fmt.Errorf("empty redis reply")
	}
	if line[0] == '-' {
		return "", false, fmt.Errorf("redis error: %s", strings.TrimSpace(line[1:]))
	}
	if line[0] != '$' {
		return "", false, fmt.Errorf("unexpected redis reply: %q", line)
	}
	sizeText := strings.TrimSpace(line[1:])
	size, err := strconv.Atoi(sizeText)
	if err != nil {
		return "", false, fmt.Errorf("invalid bulk size: %w", err)
	}
	if size == -1 {
		return "", false, nil
	}
	buf := make([]byte, size+2)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return "", false, err
	}
	return string(buf[:size]), true, nil
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
