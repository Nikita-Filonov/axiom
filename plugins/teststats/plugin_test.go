package teststats_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
	"github.com/stretchr/testify/assert"
)

func TestPlugin_RecordsPassedCase(t *testing.T) {
	stats := teststats.NewStats()
	plugin := teststats.Plugin(stats)

	cfg := &axiom.Config{
		ID:   "id1",
		Name: "case1",
		Meta: axiom.Meta{},
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{},
			AfterTest:  []axiom.TestHook{},
		},
	}

	plugin(cfg)

	for _, h := range cfg.Hooks.BeforeTest {
		h(cfg)
	}

	cfg.SubT = &testing.T{}

	for _, h := range cfg.Hooks.AfterTest {
		h(cfg)
	}

	assert.Equal(t, 1, stats.Total)
	assert.Equal(t, 1, stats.Passed)
	assert.Len(t, stats.Cases, 1)
	assert.Equal(t, "case1", stats.Cases[0].Name)
	assert.Equal(t, teststats.StatusPassed, stats.Cases[0].Status)
}

func TestPlugin_RecordsFailedCase(t *testing.T) {
	stats := teststats.NewStats()
	plugin := teststats.Plugin(stats)

	cfg := &axiom.Config{
		ID:   "id2",
		Name: "case2",
		Meta: axiom.Meta{},
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{},
			AfterTest:  []axiom.TestHook{},
		},
	}

	plugin(cfg)

	for _, h := range cfg.Hooks.BeforeTest {
		h(cfg)
	}

	fakeT := &testing.T{}
	fakeT.Fail()
	cfg.SubT = fakeT

	for _, h := range cfg.Hooks.AfterTest {
		h(cfg)
	}

	assert.Equal(t, 1, stats.Failed)
	assert.Equal(t, teststats.StatusFailed, stats.Cases[0].Status)
}

func TestPlugin_RecordsSkippedCase(t *testing.T) {
	stats := teststats.NewStats()
	plugin := teststats.Plugin(stats)

	cfg := &axiom.Config{
		ID:   "id3",
		Name: "case3",
		Skip: axiom.Skip{Enabled: true},
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{},
			AfterTest:  []axiom.TestHook{},
		},
	}

	plugin(cfg)

	for _, h := range cfg.Hooks.BeforeTest {
		h(cfg)
	}

	cfg.SubT = &testing.T{}

	for _, h := range cfg.Hooks.AfterTest {
		h(cfg)
	}

	assert.Equal(t, 1, stats.Skipped)
	assert.Equal(t, teststats.StatusSkipped, stats.Cases[0].Status)
}

func TestPlugin_RecordsFlakyCase(t *testing.T) {
	stats := teststats.NewStats()
	plugin := teststats.Plugin(stats)

	cfg := &axiom.Config{
		ID:   "id4",
		Name: "case4",
		Meta: axiom.Meta{},
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{},
			AfterTest:  []axiom.TestHook{},
		},
	}

	plugin(cfg)

	for _, h := range cfg.Hooks.BeforeTest {
		h(cfg)
		h(cfg)
	}

	cfg.SubT = &testing.T{}

	for _, h := range cfg.Hooks.AfterTest {
		h(cfg)
	}

	assert.Equal(t, 1, stats.Flaky)
	assert.Equal(t, teststats.StatusFlaky, stats.Cases[0].Status)
}
