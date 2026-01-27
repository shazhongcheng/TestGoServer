// internal/service/modules/login/login.go
package login

const (
	MsgLoginReq  = 1001
	MsgLoginResp = 1002
)

type Module struct {
	svc *LoginService
}

func (m *Module) Name() string { return "login" }
func (m *Module) Init() error {
	if m.svc == nil {
		m.svc = NewLoginService(nil, nil)
	}
	return nil
}

func NewModule(svc *LoginService) *Module {
	return &Module{svc: svc}
}
