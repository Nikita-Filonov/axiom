package testleaks

import "github.com/Nikita-Filonov/axiom"

const labelKey = "github.com/Nikita-Filonov/axiom/plugins/testleaks.attempt"

var stateKey = axiom.NewLocalKey[*attemptState]("testleaks.attempt")

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
		if _, installed := axiom.GetLocal(cfg, stateKey); installed {
			return
		}

		state := newAttemptState()
		axiom.SetLocal(cfg, stateKey, state)
		cfg.Runtime.EmitTestWrap(wrapAttempt(state, c))
	}
}
