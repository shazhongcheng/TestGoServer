package protocol

import "errors"

type ErrorCode int32

const (
	OK ErrorCode = 0

	// ---- 通用 ----
	ErrUnknown         ErrorCode = 1000
	ErrInvalidParam    ErrorCode = 1001
	ErrUnauthorized    ErrorCode = 1002
	ErrInvalidToken    ErrorCode = 1003
	ErrSessionExpired  ErrorCode = 1004
	ErrUnknownPlatform ErrorCode = 10005

	// ---- Login ----
	ErrLoginFailed ErrorCode = 1100

	// ---- Game ----
	ErrPlayerNotReady ErrorCode = 2000
)

var (
	InternalErrInvalidSession   = errors.New("invalid session")
	InternalErrInvalidToken     = errors.New("invalid token")
	InternalErrConnClosed       = errors.New("connection closed")
	InternalErrRemoteNotReady   = errors.New("remote not ready")
	InternalErrNoGateConnection = errors.New("no gate connection")
	InternalErrUnknownPlatForm  = errors.New("unknown platform")

	ErrNoGateConnection           = errors.New("gate connection not ready")
	InternalErrGameRouterNotReady = errors.New("game router not ready")
)

const (
	PlatformTest    int32 = 0
	PlatformAndroid int32 = 1
	PlatformIOS     int32 = 2
	PlatformPC      int32 = 3
)
