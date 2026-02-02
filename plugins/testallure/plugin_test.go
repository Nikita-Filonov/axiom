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

func TestPlugin_AddsSetupWrapAndCallsNext(t *testing.T) {
	cfg := &axiom.Config{SubT: t}
	p := testallure.Plugin()
	p(cfg)

	assert.Len(t, cfg.Runtime.TestWraps, 1)
	assert.Len(t, cfg.Runtime.SetupWraps, 1)

	test := cfg.Runtime.TestWraps[0](func(c *axiom.Config) {})
	test(cfg)

	called := false
	setup := cfg.Runtime.SetupWraps[0]("setup", func() {
		called = true
	})
	setup()

	assert.True(t, called, "setup next() must be called")
}

func TestPlugin_AddsTeardownWrapAndCallsNext(t *testing.T) {
	cfg := &axiom.Config{SubT: t}
	p := testallure.Plugin()
	p(cfg)

	assert.Len(t, cfg.Runtime.TestWraps, 1)
	assert.Len(t, cfg.Runtime.TeardownWraps, 1)

	test := cfg.Runtime.TestWraps[0](func(c *axiom.Config) {})
	test(cfg)

	called := false
	td := cfg.Runtime.TeardownWraps[0]("teardown", func() {
		called = true
	})
	td()

	assert.True(t, called, "teardown next() must be called")
}

func TestPlugin_DoesNotPanic_WhenSubTIsNil(t *testing.T) {
	cfg := &axiom.Config{} // SubT == nil
	p := testallure.Plugin()

	assert.NotPanics(t, func() { p(cfg) })
}

func TestPlugin_AddsAllExpectedRuntimeHooks(t *testing.T) {
	cfg := &axiom.Config{SubT: t}
	p := testallure.Plugin()

	p(cfg)

	assert.Len(t, cfg.Runtime.TestWraps, 1)
	assert.Len(t, cfg.Runtime.StepWraps, 1)
	assert.Len(t, cfg.Runtime.SetupWraps, 1)
	assert.Len(t, cfg.Runtime.TeardownWraps, 1)
	assert.Len(t, cfg.Runtime.ArtefactSinks, 1)
}
