package selflog

import (
	stdlog "log"
	"os"
)

type StdLogger struct {
	prefix string
	logger *stdlog.Logger
}

// NewStdLogger 创建一个基于标准库的 Logger
func NewStdLogger(prefix string) Logger {
	return &StdLogger{
		prefix: prefix,
		logger: stdlog.New(os.Stdout, "", stdlog.LstdFlags),
	}
}

func (l *StdLogger) Debug(msg string, args ...any) {
	l.printf("DEBUG", msg, args...)
}

func (l *StdLogger) Info(msg string, args ...any) {
	l.printf("INFO", msg, args...)
}

func (l *StdLogger) Warn(msg string, args ...any) {
	l.printf("WARN", msg, args...)
}

func (l *StdLogger) Error(msg string, args ...any) {
	l.printf("ERROR", msg, args...)
}

func (l *StdLogger) printf(level, msg string, args ...any) {
	full := "[" + level + "]"
	if l.prefix != "" {
		full += "[" + l.prefix + "]"
	}
	full += " " + msg

	l.logger.Printf(full, args...)
}
