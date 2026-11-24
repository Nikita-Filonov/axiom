package testleaks

import (
	"strings"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestWrapAttemptWithoutTestingT(t *testing.T) {
	called := false
	wrapped := wrapAttempt(newAttemptState(), Config{})(func(*axiom.Config) { called = true })
	wrapped(&axiom.Config{Attempt: axiom.Attempt{Decision: axiom.Decision{Kind: axiom.DecisionRun}}})
	assert.True(t, called, "wrapped action was not called")
}

func TestWrapAttemptWithoutParentContext(t *testing.T) {
	t.Run("attempt", func(t *testing.T) {
		cfg := &axiom.Config{
			RootT:   t,
			Attempt: axiom.Attempt{Decision: axiom.Decision{Kind: axiom.DecisionRun}},
		}
		wrapped := wrapAttempt(newAttemptState(), Config{})(func(current *axiom.Config) {
			assert.Same(t, cfg, current, "wrapped action received a different config")
		})
		wrapped(cfg)
	})
}

func TestAttemptID(t *testing.T) {
	cfg := &axiom.Config{}
	first := attemptID(cfg)
	second := attemptID(cfg)
	assert.NotEmpty(t, first)
	assert.NotEmpty(t, second)
	assert.NotEqual(t, first, second, "random attempt IDs are not distinct")
	cfg.Attempt.RunID = "execution"
	cfg.Attempt.Number = 2
	assert.Equal(t, "execution/2", attemptID(cfg))
	assert.True(t, strings.HasPrefix(labelKey, "github.com/Nikita-Filonov/axiom/plugins/testleaks"), "pprof label key is not namespaced")
}
