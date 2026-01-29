// internal/service/dispatcher.go
package service

import (
	"fmt"
	"go.uber.org/zap"
	"runtime/debug"
)

type Dispatcher struct {
	registry *Registry
	logger   *zap.Logger
}

func NewDispatcher(reg *Registry, logger *zap.Logger) *Dispatcher {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &Dispatcher{registry: reg, logger: logger}
}

func (d *Dispatcher) Dispatch(ctx *Context) {
	defer func() {
		if r := recover(); r != nil {
			d.logger.Error("handler panic",
				zap.Any("panic", r),
				zap.String("panic_type", fmt.Sprintf("%T", r)),
				zap.ByteString("stack", debug.Stack()),
				zap.Int("msg_id", ctx.MsgID),
				zap.Int64("session", ctx.SessionID),
				zap.Int64("player", ctx.PlayerID),
				zap.Int64("conn_id", 0),
				zap.String("trace_id", ctx.TraceID),
			)
		}
	}()

	handler, ok := d.registry.GetHandler(ctx.MsgID)
	if !ok {
		d.logger.Warn("no handler for msgID",
			zap.Int("msg_id", ctx.MsgID),
			zap.Int64("session", ctx.SessionID),
			zap.Int64("player", ctx.PlayerID),
			zap.String("reason", "handler_not_found"),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ctx.TraceID),
		)
		return
	}

	if err := handler(ctx); err != nil {
		d.logger.Warn("handler error",
			zap.Int("msg_id", ctx.MsgID),
			zap.Int64("session", ctx.SessionID),
			zap.Int64("player", ctx.PlayerID),
			zap.String("reason", err.Error()),
			zap.Int64("conn_id", 0),
			zap.String("trace_id", ctx.TraceID),
		)
	}
}
