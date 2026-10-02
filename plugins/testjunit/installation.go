package testjunit

import "github.com/Nikita-Filonov/axiom"

var installedKey = axiom.NewLocalKey[map[*Reporter]bool]("github.com/Nikita-Filonov/axiom/plugins/testjunit.installed")

// markInstalled reports whether reporter was not yet installed on cfg.
func markInstalled(cfg *axiom.Config, reporter *Reporter) bool {
	installed, _ := axiom.GetLocal(cfg, installedKey)
	if installed[reporter] {
		return false
	}
	if installed == nil {
		installed = make(map[*Reporter]bool)
		axiom.SetLocal(cfg, installedKey, installed)
	}
	installed[reporter] = true
	return true
}
