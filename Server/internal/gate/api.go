// internal/gate/api.go
package gate

type Gateway interface {
	Reply(sessionID int64, msgID int, data []byte) error
	Push(sessionID int64, msgID int, data []byte) error
	Kick(sessionID int64, reason string) error
}
