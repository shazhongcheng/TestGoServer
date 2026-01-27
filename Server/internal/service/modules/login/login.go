// internal/service/modules/login/login.go
package login

const (
	MsgLoginReq  = 1001
	MsgLoginResp = 1002
)

type Module struct{}

func (m *Module) Name() string { return "login" }
func (m *Module) Init() error  { return nil }
