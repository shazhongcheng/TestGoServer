package player

import "fmt"

const (
	accountRoleKeyPrefix   = "account:role:"
	playerProfileKeyPrefix = "player:profile:"
)

type PlayerProfile struct {
	RoleID    int64  `json:"role_id"`
	AccountID string `json:"account_id,omitempty"`
	NickName  string `json:"nickname"`
	Level     int32  `json:"level"`
	Exp       int64  `json:"exp"`
	Gold      int64  `json:"gold"`
	Stamina   int64  `json:"stamina"`
}

func AccountRoleKey(accountID string) string {
	return fmt.Sprintf("%s%s", accountRoleKeyPrefix, accountID)
}

func PlayerProfileKey(roleID int64) string {
	return fmt.Sprintf("%s%d", playerProfileKeyPrefix, roleID)
}

func NewProfile(roleID int64, accountID string) PlayerProfile {
	return PlayerProfile{
		RoleID:    roleID,
		AccountID: accountID,
		NickName:  fmt.Sprintf("Player-%d", roleID),
		Level:     1,
		Exp:       0,
		Gold:      100,
		Stamina:   100,
	}
}
