package login

import (
	"context"

	"game-server/internal/player"
)

type UIDGenerator interface {
	NextUID(ctx context.Context) (int64, error)
}

type LoginService struct {
	uidGen UIDGenerator
	store  player.Store
}

func NewLoginService(uidGen UIDGenerator, store player.Store) *LoginService {
	return &LoginService{uidGen: uidGen, store: store}
}

func (s *LoginService) NextUID(ctx context.Context) (int64, error) {
	if s.uidGen == nil {
		return 0, nil
	}
	return s.uidGen.NextUID(ctx)
}

func (s *LoginService) ResolveRoleID(ctx context.Context, accountID string) (int64, bool, error) {
	if s.store == nil {
		roleID, err := s.NextUID(ctx)
		return roleID, false, err
	}
	roleID, ok, err := s.store.LoadRoleID(ctx, accountID)
	if err != nil {
		return 0, false, err
	}
	if ok {
		return roleID, true, nil
	}
	roleID, err = s.NextUID(ctx)
	if err != nil {
		return 0, false, err
	}
	profile := player.NewProfile(roleID, accountID)
	if err := s.store.SaveRoleID(ctx, accountID, roleID); err != nil {
		return 0, false, err
	}
	if err := s.store.SaveProfile(ctx, &profile); err != nil {
		return 0, false, err
	}
	return roleID, false, nil
}
