package login

import "context"

type UIDGenerator interface {
	NextUID(ctx context.Context) (int64, error)
}

type LoginService struct {
	uidGen UIDGenerator
}

func NewLoginService(uidGen UIDGenerator) *LoginService {
	return &LoginService{uidGen: uidGen}
}

func (s *LoginService) NextUID(ctx context.Context) (int64, error) {
	if s.uidGen == nil {
		return 0, nil
	}
	return s.uidGen.NextUID(ctx)
}
