package teststats

import "github.com/Nikita-Filonov/axiom"

var installedKey = axiom.NewLocalKey[map[*Stats]bool]("teststats.installed")

// Plugin records an [Attempt] in stats for every attempt that starts or is
// skipped by policy. The attempt is recorded when its testing.T finishes:
// after hooks, fixture cleanup, child subtests, and t.Cleanup callbacks.
// Installing the same stats more than once on a Config, for example on both
// the Runner and the Case, records each attempt once. Plugin panics if stats
// is nil.
func Plugin(stats *Stats) axiom.Plugin {
	if stats == nil {
		panic("teststats: nil stats")
	}

	return func(cfg *axiom.Config) {
		if markInstalled(cfg, stats) {
			r := &recorder{cfg: cfg, stats: stats}
			cfg.Runtime.EmitEventSink(r.observe)
		}
	}
}

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
