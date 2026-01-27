// internal/router/router.go
package router

var routes = make(map[int]RouteRule)

func Init() {
	for _, r := range routeTable {
		routes[r.MsgID] = r
	}
}

func GetRoute(msgID int) (RouteRule, bool) {
	r, ok := routes[msgID]
	return r, ok
}
