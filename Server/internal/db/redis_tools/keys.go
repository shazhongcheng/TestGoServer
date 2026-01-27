package redis_tools

import (
	"fmt"
	"strconv"
)

const (
	KeyUIDNext = "uid:next"
)

func KeyPlayerBase(playerID int64) string {
	return fmt.Sprintf("player:%d:base", playerID)
}

func KeyRank(rankName string) string {
	return fmt.Sprintf("rank:%s", rankName)
}

func AccountRoleKey(accountID string) string {
	return "account:" + accountID + ":role"
}

func PlayerProfileKey(roleID int64) string {
	return "player:" + strconv.FormatInt(roleID, 10) + ":profile"
}
