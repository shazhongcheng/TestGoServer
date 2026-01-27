// internal/protocol/msgid.go
package protocol

// =======================
// Gate / Framework
// =======================
const (
	// Gate / Framework
	MsgGateBegin   = 0
	MsgResumeReq   = 1
	MsgResumeRsp   = 2
	MsgSessionInit = 3

	MsgHeartbeatReq = 10
	MsgHeartbeatRsp = 11

	MsgErrorRsp = 21

	MsgGateEnd = 1000
)

// =======================
// Login / Account (Service)
// =======================
const (
	// Login / Account
	MsgLoginBegin = 1000
	MsgLoginReq   = 1001
	MsgLoginRsp   = 1002
	MsgLoginEnd   = 2000
)

const (
	// Chat / Social
	MsgChatBegin   = 2000
	MsgChatSendReq = 2001
	MsgChatSendRsp = 2002
	MsgChatEnd     = 3000
)

// =======================
// Game Logic (Game)
// =======================
const (
	// Game Logic
	MsgGameBegin           = 3000
	MsgPlayerEnterGameReq  = 3001
	MsgPlayerEnterGameRsp  = 3002
	MsgLoadPlayerDataReq   = 3003
	MsgLoadPlayerDataRsp   = 3004
	MsgPlayerResumeReq     = 3005
	MsgPlayerOfflineNotify = 3006
	//MsgPlayerReEnterGameReq = 3007
	MsgGameEnd = 4000
)
