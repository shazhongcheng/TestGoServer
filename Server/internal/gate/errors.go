package gate

type ErrorCode int32

const (
	OK ErrorCode = 0

	// ---- 通用 ----
	ErrUnknown        ErrorCode = 1000
	ErrInvalidParam   ErrorCode = 1001
	ErrUnauthorized   ErrorCode = 1002
	ErrInvalidToken   ErrorCode = 1003
	ErrSessionExpired ErrorCode = 1004

	// ---- Login ----
	ErrLoginFailed ErrorCode = 1100

	// ---- Game ----
	ErrPlayerNotReady ErrorCode = 2000
)
