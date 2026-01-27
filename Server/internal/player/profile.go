package player

import "fmt"

const (
	accountKeyPrefix = "account:"
	playerKeyPrefix  = "player:"
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

// ======================
// Redis Keys
// ======================
func AccountRoleKey(accountID string) string {
	return fmt.Sprintf("%s%s:role", accountKeyPrefix, accountID)
}

func PlayerProfileKey(roleID int64) string {
	return fmt.Sprintf("%s%d:profile", playerKeyPrefix, roleID)
}

// ======================
// Factory
// ======================
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
