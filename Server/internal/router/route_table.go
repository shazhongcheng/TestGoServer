// internal/router/route_table.go
package router

type TargetType int

const (
	TargetService TargetType = iota
	TargetGame
	TargetDB
)

type RouteRule struct {
	Target TargetType
	Module string // service 模块名
}

var routeTable = []RouteRule{
	{Target: TargetService, Module: "login"},
	{Target: TargetService, Module: "chat"},
	{Target: TargetGame, Module: ""},
}
