package transport

import "game-server/internal/protocol/internalpb"

type Conn interface {
	ReadEnvelope() (*internalpb.Envelope, error)
	WriteEnvelope(*internalpb.Envelope) error
	Close() error
}
