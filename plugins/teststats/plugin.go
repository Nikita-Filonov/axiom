package teststats

import (
	"github.com/Nikita-Filonov/axiom"
)

func Plugin(stats *Stats) axiom.Plugin {
	return func(cfg *axiom.Config) {
		result := NewCaseResult(cfg)
		attempts := 0

		cfg.Hooks.BeforeTest = append(
			cfg.Hooks.BeforeTest,
			func(_ *axiom.Config) { attempts++ },
		)

		cfg.Hooks.AfterTest = append(
			cfg.Hooks.AfterTest,
			func(c *axiom.Config) {
				result.Finalize(c, attempts)
				stats.Record(result)
			},
		)
	}
}
