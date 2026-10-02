package testleaks

import "github.com/Nikita-Filonov/axiom"

// Plugin checks one running attempt after the body and its later-registered
// testing.T cleanups. The planning Config does not run a test and is ignored.
// Applying Plugin twice to the same Config installs one checker. A nil Config
// panics.
func Plugin(options ...Option) axiom.Plugin {
	c := newConfig(options...)

	return func(cfg *axiom.Config) {
		if cfg == nil {
			panic("testleaks: nil config")
		}
		if cfg.Execution.ID != "" && cfg.Execution.Attempt == 0 {
			return
		}
		state, installed := installAttempt(cfg)
		if !installed {
			return
		}
		cfg.Runtime.EmitTestWrap(wrapAttempt(state, c))
	}
}
