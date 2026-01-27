package player

import "fmt"

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
