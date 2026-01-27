package selflog

type NopLogger struct{}

func NewNopLogger() Logger {
	return &NopLogger{}
}

func (l *NopLogger) Debug(string, ...any) {}
func (l *NopLogger) Info(string, ...any)  {}
func (l *NopLogger) Warn(string, ...any)  {}
func (l *NopLogger) Error(string, ...any) {}
