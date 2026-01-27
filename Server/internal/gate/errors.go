package gate

import "errors"

var (
	ErrInvalidSession = errors.New("invalid session")
	ErrInvalidToken   = errors.New("invalid token")
	ErrConnClosed     = errors.New("connection closed")
)
