package testleaks

import "github.com/Nikita-Filonov/axiom"

var stateKey = axiom.NewLocalKey[*attemptState]("github.com/Nikita-Filonov/axiom/plugins/testleaks.attempt")

// installAttempt returns a new attempt state, or nil and false if already installed.
func installAttempt(cfg *axiom.Config) (*attemptState, bool) {
	if _, installed := axiom.GetLocal(cfg, stateKey); installed {
		return nil, false
	}
	state := newAttemptState()
	axiom.SetLocal(cfg, stateKey, state)
	return state, true
}
