package player_module

// game/player/module_registry.go

var registeredModules []func() Module

func RegisterModule(f func() Module) {
	registeredModules = append(registeredModules, f)
}

func CreateModules() []Module {
	var ms []Module
	for _, f := range registeredModules {
		ms = append(ms, f())
	}
	return ms
}
