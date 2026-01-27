// internal/protocol/msgid.go
package protocol

const (
	// ----- Gate Control -----
	MsgResumeReq = 1
	MsgResumeRsp = 2

	MsgHeartbeatReq = 10
	MsgHeartbeatRsp = 11

	// ----- Business -----
	MsgLoginReq = 1001
	MsgLoginRsp = 1002
	MsgChatReq  = 2001
)
