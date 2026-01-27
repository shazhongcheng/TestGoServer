package redis_tools

import (
	"fmt"
	"strconv"
)

const (
	KeyUIDNext = "uid:next"

	keyAccountPrefix = "account:"
	keyPlayerPrefix  = "player:"
	keyRankPrefix    = "rank:"
)

func KeyPlayerBase(playerID int64) string {
	return fmt.Sprintf("%s%d:base", keyPlayerPrefix, playerID)
}

func KeyRank(rankName string) string {
	return fmt.Sprintf("%s%s", keyRankPrefix, rankName)
}

func AccountRoleKey(accountID string) string {
	return fmt.Sprintf("%s%s:role", keyAccountPrefix, accountID)
}

func PlayerProfileKey(roleID int64) string {
	return fmt.Sprintf("%s%s:profile", keyPlayerPrefix, strconv.FormatInt(roleID, 10))
}
