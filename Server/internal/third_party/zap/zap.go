package zap

import (
	"fmt"
	stdlog "log"
	"strings"
)

type Field struct {
	Key   string
	Value any
}

type Logger struct {
	name    string
	enabled bool
}

type Config struct{}

func NewProductionConfig() Config {
	return Config{}
}

func (c Config) Build() (*Logger, error) {
	return &Logger{enabled: true}, nil
}

func NewNop() *Logger {
	return &Logger{enabled: false}
}

func (l *Logger) Named(name string) *Logger {
	if l == nil {
		return &Logger{name: name, enabled: false}
	}
	return &Logger{name: name, enabled: l.enabled}
}

func (l *Logger) Sync() error {
	return nil
}

func (l *Logger) Debug(msg string, fields ...Field) {
	l.log("DEBUG", msg, fields...)
}

func (l *Logger) Info(msg string, fields ...Field) {
	l.log("INFO", msg, fields...)
}

func (l *Logger) Warn(msg string, fields ...Field) {
	l.log("WARN", msg, fields...)
}

func (l *Logger) Error(msg string, fields ...Field) {
	l.log("ERROR", msg, fields...)
}

func (l *Logger) log(level, msg string, fields ...Field) {
	if l == nil || !l.enabled {
		return
	}
	var sb strings.Builder
	sb.WriteString(level)
	if l.name != "" {
		sb.WriteString("[" + l.name + "]")
	}
	sb.WriteString(" ")
	sb.WriteString(msg)
	for _, field := range fields {
		sb.WriteString(" ")
		sb.WriteString(field.Key)
		sb.WriteString("=")
		sb.WriteString(fmt.Sprint(field.Value))
	}
	stdlog.Print(sb.String())
}

func Int(key string, val int) Field {
	return Field{Key: key, Value: val}
}

func Int64(key string, val int64) Field {
	return Field{Key: key, Value: val}
}

func String(key, val string) Field {
	return Field{Key: key, Value: val}
}

func Err(key string, err error) Field {
	return Field{Key: key, Value: err}
}

func Error(err error) Field {
	return Field{Key: "Error", Value: err}
}

func Any(key string, val interface{}) Field {
	return Field{
		Key:   key,
		Value: val,
	}
}

func ByteString(key string, val []byte) Field {
	return Field{
		Key:   key,
		Value: string(val),
	}
}
