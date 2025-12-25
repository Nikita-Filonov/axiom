package testassert

import (
	"github.com/Nikita-Filonov/axiom"
)

func Plugin() axiom.Plugin {
	return func(cfg *axiom.Config) {
		cfg.Runtime.EmitAssertSink(func(a axiom.Assert) {
			if cfg.SubT == nil {
				return
			}

			HandleAssert(cfg.SubT, a)
		})
	}
}
