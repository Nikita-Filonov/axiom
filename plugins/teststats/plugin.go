package teststats

import (
	"github.com/Nikita-Filonov/axiom"
)

func Plugin(stats *Stats) axiom.Plugin {
	return func(cfg *axiom.Config) {
		result := NewCaseResult(cfg)
		attempts := 0

		cfg.Hooks.BeforeSubTest = append(
			cfg.Hooks.BeforeSubTest,
			func(_ *axiom.Config) { attempts++ },
		)

		cfg.Hooks.AfterSubTest = append(
			cfg.Hooks.AfterSubTest,
			func(c *axiom.Config) {
				result.Finalize(c, attempts)
				stats.Record(result)
			},
		)
	}
}
