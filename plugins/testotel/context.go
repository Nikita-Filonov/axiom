package testotel

import (
	"context"

	"github.com/Nikita-Filonov/axiom"
)

// Context returns the active attempt span context. Before an attempt starts,
// or without the plugin, it returns cfg.Context.Raw (or Background if unset).
// Pass this context to instrumented clients to make their spans children of
// the test attempt. Context panics for a nil Config.
func Context(cfg *axiom.Config) context.Context {
	if cfg == nil {
		panic("testotel: nil config")
	}
	if a, ok := axiom.GetLocal(cfg, attemptKey); ok {
		if ctx := a.context(); ctx != nil {
			return ctx
		}
	}
	if cfg.Context.Raw != nil {
		return cfg.Context.Raw
	}
	return context.Background()
}
