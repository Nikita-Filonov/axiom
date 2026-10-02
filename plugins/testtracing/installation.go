package testtracing

import "github.com/Nikita-Filonov/axiom"

var installedKey = axiom.NewLocalKey[map[*Trace]bool]("github.com/Nikita-Filonov/axiom/plugins/testtracing.installed")

// markInstalled reports whether trace was not yet installed on cfg.
func markInstalled(cfg *axiom.Config, trace *Trace) bool {
	installed, _ := axiom.GetLocal(cfg, installedKey)
	if installed[trace] {
		return false
	}
	if installed == nil {
		installed = make(map[*Trace]bool)
		axiom.SetLocal(cfg, installedKey, installed)
	}
	installed[trace] = true
	return true
}
