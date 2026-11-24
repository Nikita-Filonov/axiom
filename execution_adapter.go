package axiom

import (
	"testing"

	"github.com/Nikita-Filonov/axiom/internal/execution"
)

// runCase is the single entry from Axiom into the execution engine.
func (r *Runner) runCase(t *testing.T, testCase Case, action TestAction) {
	execution.Run(t, r.plan(t, testCase, action))
}

// plan resolves scheduling from a planning Config and prepares fresh attempt
// Configs lazily when the engine requests them.
func (r *Runner) plan(t *testing.T, testCase Case, action TestAction) execution.Plan {
	template := testCase.Copy()
	base := newAttempt()
	planning := r.prepare(t, template, base)

	return execution.Plan{
		Name:        planning.Case.Name,
		MaxAttempts: planning.Retry.Times,
		RetryDelay:  planning.Retry.Delay,
		Parallel:    planning.Parallel.Enabled,

		Prepare: func(ctx execution.PrepareContext) execution.Spec {
			cfg := r.prepare(t, template, Attempt{
				RunID:  base.RunID,
				Number: ctx.Number,
			})

			return cfg.spec(action)
		},
	}
}

// prepare builds one isolated Config, installs plugins, and freezes the engine
// decision. Exclude is the only plugin decision that overrides merged Skip.
func (r *Runner) prepare(t *testing.T, template Case, attempt Attempt) *Config {
	testCase := template.Copy()
	cfg := r.buildConfig(t, &testCase, attempt)
	cfg.applyPlugins()
	cfg.applyDecision()

	return cfg
}

// spec maps one prepared Config to the engine callback contract. The engine
// never sees a Config, Runner, Case, fixture registry, or plugin.
func (c *Config) spec(action TestAction) execution.Spec {
	current := c
	attempt := c.Attempt

	name := ""
	if c.Case != nil {
		name = c.Case.Name
	}

	return execution.Spec{
		Name: name,
		Decision: execution.Decision{
			Kind:   execution.DecisionKind(attempt.Decision.Kind),
			Reason: attempt.Decision.Reason,
		},
		Parallel: c.Parallel.Enabled,

		Bind: func(t *testing.T) { c.SubT = t },

		Around: func(next func()) {
			c.Runtime.test(c, func(cfg *Config) {
				current = cfg
				next()
			})
		},

		Before:  func() { current.Hooks.applyBeforeTest(current) },
		Body:    func() { action(current) },
		After:   func() { current.Hooks.applyAfterTest(current) },
		Release: func() { current.Fixtures.teardown(current) },

		Observe: func(event execution.Event) {
			c.observe(attempt, event)
		},
	}
}
