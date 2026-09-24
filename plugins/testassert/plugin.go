package testassert

import (
	"github.com/Nikita-Filonov/axiom"
)

// Plugin evaluates emitted assertion facts with testify/assert.
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
