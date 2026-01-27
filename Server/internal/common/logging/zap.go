package logging

import "go.uber.org/zap"

func NewLogger(name string) (*zap.Logger, error) {
	cfg := zap.NewProductionConfig()
	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}
	if name != "" {
		logger = logger.Named(name)
	}
	return logger, nil
}
