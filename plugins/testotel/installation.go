package testotel

import (
	"github.com/Nikita-Filonov/axiom"
	"go.opentelemetry.io/otel/trace"
)

var attemptKey = axiom.NewLocalKey[*attempt]("github.com/Nikita-Filonov/axiom/plugins/testotel.attempt")

// installAttempt returns a new attempt, or nil and false if already installed.
func installAttempt(cfg *axiom.Config, tracer trace.Tracer) (*attempt, bool) {
	if _, installed := axiom.GetLocal(cfg, attemptKey); installed {
		return nil, false
	}
	a := &attempt{cfg: cfg, tracer: tracer}
	axiom.SetLocal(cfg, attemptKey, a)
	return a, true
}
