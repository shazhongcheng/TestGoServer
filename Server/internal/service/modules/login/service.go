package login

type LoginService struct {
	// repo / redis / config
}

func NewLoginService() *LoginService {
	return &LoginService{}
}

//func (s *LoginService) Login(req *LoginReq) (*LoginResult, error) {
//	// 账号校验 / token / 风控
//	return &LoginResult{}, nil
//}
