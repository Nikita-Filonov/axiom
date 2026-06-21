package testexplain

import "github.com/Nikita-Filonov/axiom"

func Plugin(explainer *Explainer) axiom.Plugin {
	return func(cfg *axiom.Config) {
		cfg.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				explainer.Record(ExplainConfig(c))
				next(c)
			}
		})
	}
}
