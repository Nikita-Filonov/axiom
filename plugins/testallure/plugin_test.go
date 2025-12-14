package testallure_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
	"github.com/stretchr/testify/assert"
)

func TestPlugin_AddsTestWrapAndCallsNext(t *testing.T) {
	cfg := &axiom.Config{SubT: t}
	p := testallure.Plugin()

	p(cfg)

	assert.Len(t, cfg.Runtime.TestWraps, 1)

	called := false

	wrapped := cfg.Runtime.TestWraps[0](func(c *axiom.Config) {
		called = true
	})

	wrapped(cfg)

	assert.True(t, called, "next() must be called")
}

func TestPlugin_AddsStepWrapAndCallsNext(t *testing.T) {
	cfg := &axiom.Config{}
	p := testallure.Plugin()

	p(cfg)

	assert.Len(t, cfg.Runtime.StepWraps, 1)

	called := false

	wrapped := cfg.Runtime.StepWraps[0]("step-name", func() {
		called = true
	})

	wrapped()

	assert.True(t, called, "step next() must be called")
}

func TestPlugin_AddsArtefactSink(t *testing.T) {
	cfg := &axiom.Config{}
	p := testallure.Plugin()

	p(cfg)

	assert.Len(t, cfg.Runtime.ArtefactSinks, 1)
}
