// internal/router/router.go
package router

import "game-server/internal/protocol"

var routes = make(map[int]RouteRule)

//func Init() {
//	for _, r := range routeTable {
//		routes[r.MsgID] = r
//	}
//}

func GetRoute(msgID int) (RouteRule, bool) {
	switch {
	case msgID >= protocol.MsgLoginBegin && msgID < protocol.MsgLoginEnd:
		return RouteRule{Target: TargetService, Module: "login"}, true

	case msgID >= protocol.MsgChatBegin && msgID < protocol.MsgChatEnd:
		return RouteRule{Target: TargetService, Module: "chat"}, true

	case msgID >= protocol.MsgGameBegin && msgID < protocol.MsgGameEnd:
		return RouteRule{Target: TargetGame}, true

	default:
		return RouteRule{}, false
	}
}
