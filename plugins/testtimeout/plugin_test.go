package testtimeout

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestPlugin_NoTimeout_DoesNotWrap(t *testing.T) {
	cfg := &axiom.Config{}

	Plugin(WithContextDeadline())(cfg)

	assert.Empty(t, cfg.Runtime.TestWraps, "a non-positive timeout must be a no-op")
}

func TestPlugin_WithTimeout_AddsWrap(t *testing.T) {
	cfg := &axiom.Config{}

	Plugin(WithTimeout(time.Second))(cfg)

	assert.Len(t, cfg.Runtime.TestWraps, 1)
}

func TestPlugin_PassingCaseRunsBody(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(Plugin(WithTimeout(5 * time.Second))),
	)

	ran := false
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("fast")), func(cfg *axiom.Config) {
		ran = true
	})

	assert.True(t, ran, "a case that finishes within the budget must run normally")
}
