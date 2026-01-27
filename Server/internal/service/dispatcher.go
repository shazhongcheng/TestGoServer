// internal/service/dispatcher.go
package service

import "log"

type Dispatcher struct {
	registry *Registry
}

func NewDispatcher(reg *Registry) *Dispatcher {
	return &Dispatcher{registry: reg}
}

func (d *Dispatcher) Dispatch(ctx *Context) {
	handler, ok := d.registry.GetHandler(ctx.MsgID)
	if !ok {
		log.Printf("[Service] no handler for msgID=%d", ctx.MsgID)
		return
	}

	if err := handler(ctx); err != nil {
		log.Printf("[Service] handler error msgID=%d err=%v", ctx.MsgID, err)
	}
}
