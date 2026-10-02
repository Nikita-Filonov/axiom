package teststats

import "github.com/Nikita-Filonov/axiom"

var installedKey = axiom.NewLocalKey[map[*Stats]bool]("github.com/Nikita-Filonov/axiom/plugins/teststats.installed")

// markInstalled reports whether stats was not yet installed on cfg.
func markInstalled(cfg *axiom.Config, stats *Stats) bool {
	installed, _ := axiom.GetLocal(cfg, installedKey)
	if installed[stats] {
		return false
	}
	if installed == nil {
		installed = make(map[*Stats]bool)
		axiom.SetLocal(cfg, installedKey, installed)
	}
	installed[stats] = true
	return true
}
