package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewHooks_WithOptions(t *testing.T) {
	var beforeCalled, afterCalled bool

	hooks := axiom.NewHooks(
		axiom.WithBeforeTest(func(cfg *axiom.Config) { beforeCalled = true }),
		axiom.WithAfterTest(func(cfg *axiom.Config) { afterCalled = true }),
	)

	assert.Len(t, hooks.BeforeTest, 1)
	assert.Len(t, hooks.AfterTest, 1)

	cfg := &axiom.Config{}
	hooks.BeforeTest[0](cfg)
	hooks.AfterTest[0](cfg)

	assert.True(t, beforeCalled)
	assert.True(t, afterCalled)
}

func TestHooks_ApplyBeforeAfterTest(t *testing.T) {
	var countBefore, countAfter int

	h := axiom.Hooks{
		BeforeTest: []axiom.TestHook{
			func(cfg *axiom.Config) { countBefore++ },
			func(cfg *axiom.Config) { countBefore++ },
		},
		AfterTest: []axiom.TestHook{
			func(cfg *axiom.Config) { countAfter++ },
		},
	}

	cfg := &axiom.Config{}

	h.ApplyBeforeTest(cfg)
	h.ApplyAfterTest(cfg)

	assert.Equal(t, 2, countBefore)
	assert.Equal(t, 1, countAfter)
}

func TestHooks_ApplyBeforeAfterStep(t *testing.T) {
	var beforeStep, afterStep []string

	h := axiom.Hooks{
		BeforeStep: []axiom.StepHook{
			func(cfg *axiom.Config, name string) { beforeStep = append(beforeStep, "A:"+name) },
			func(cfg *axiom.Config, name string) { beforeStep = append(beforeStep, "B:"+name) },
		},
		AfterStep: []axiom.StepHook{
			func(cfg *axiom.Config, name string) { afterStep = append(afterStep, "C:"+name) },
		},
	}

	cfg := &axiom.Config{}

	h.ApplyBeforeStep(cfg, "login")
	h.ApplyAfterStep(cfg, "login")

	assert.Equal(t, []string{"A:login", "B:login"}, beforeStep)
	assert.Equal(t, []string{"C:login"}, afterStep)
}

func TestHooks_Join(t *testing.T) {
	var a, b int

	h1 := axiom.Hooks{
		BeforeTest: []axiom.TestHook{func(cfg *axiom.Config) { a++ }},
		AfterTest:  []axiom.TestHook{func(cfg *axiom.Config) { b++ }},
	}

	h2 := axiom.Hooks{
		BeforeTest: []axiom.TestHook{func(cfg *axiom.Config) { a += 10 }},
		AfterTest:  []axiom.TestHook{func(cfg *axiom.Config) { b += 20 }},
	}

	merged := h1.Join(h2)

	assert.Len(t, merged.BeforeTest, 2)
	assert.Len(t, merged.AfterTest, 2)

	cfg := &axiom.Config{}

	merged.BeforeTest[0](cfg) // +1
	merged.BeforeTest[1](cfg) // +10
	assert.Equal(t, 11, a)

	merged.AfterTest[0](cfg) // +1
	merged.AfterTest[1](cfg) // +20
	assert.Equal(t, 21, b)
}

func TestNewHooks_WithStepOptions(t *testing.T) {
	var beforeCalled, afterCalled string

	hooks := axiom.NewHooks(
		axiom.WithBeforeStep(func(cfg *axiom.Config, name string) { beforeCalled = "before:" + name }),
		axiom.WithAfterStep(func(cfg *axiom.Config, name string) { afterCalled = "after:" + name }),
	)

	assert.Len(t, hooks.BeforeStep, 1)
	assert.Len(t, hooks.AfterStep, 1)

	cfg := &axiom.Config{}
	hooks.BeforeStep[0](cfg, "x")
	hooks.AfterStep[0](cfg, "x")

	assert.Equal(t, "before:x", beforeCalled)
	assert.Equal(t, "after:x", afterCalled)
}

func TestHooks_Join_Step(t *testing.T) {
	var beforeStepCount, afterStepCount int

	h1 := axiom.Hooks{
		BeforeStep: []axiom.StepHook{
			func(cfg *axiom.Config, name string) { beforeStepCount++ },
		},
		AfterStep: []axiom.StepHook{
			func(cfg *axiom.Config, name string) { afterStepCount++ },
		},
	}

	h2 := axiom.Hooks{
		BeforeStep: []axiom.StepHook{
			func(cfg *axiom.Config, name string) { beforeStepCount += 10 },
		},
		AfterStep: []axiom.StepHook{
			func(cfg *axiom.Config, name string) { afterStepCount += 20 },
		},
	}

	m := h1.Join(h2)

	assert.Len(t, m.BeforeStep, 2)
	assert.Len(t, m.AfterStep, 2)

	cfg := &axiom.Config{}

	m.BeforeStep[0](cfg, "x") // +1
	m.BeforeStep[1](cfg, "x") // +10
	assert.Equal(t, 11, beforeStepCount)

	m.AfterStep[0](cfg, "x") // +1
	m.AfterStep[1](cfg, "x") // +20
	assert.Equal(t, 21, afterStepCount)
}
