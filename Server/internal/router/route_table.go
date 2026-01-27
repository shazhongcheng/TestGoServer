// internal/router/route_table.go
package router

type TargetType int

const (
	TargetService TargetType = iota
	TargetGame
	TargetDB
)

type RouteRule struct {
	MsgID  int
	Target TargetType
	Module string // service 模块名
}

var routeTable = []RouteRule{
	{MsgID: 1001, Target: TargetService, Module: "login"},
	{MsgID: 2001, Target: TargetService, Module: "chat"},
	{MsgID: 3001, Target: TargetGame, Module: ""},
}
