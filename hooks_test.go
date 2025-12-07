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

	// simulate call
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

func TestHooks_ApplyBeforeAfterSubTest(t *testing.T) {
	var beforeCount, afterCount int

	h := axiom.Hooks{
		BeforeSubTest: []axiom.SubTestHook{
			func(cfg *axiom.Config) { beforeCount++ },
			func(cfg *axiom.Config) { beforeCount++ },
		},
		AfterSubTest: []axiom.SubTestHook{
			func(cfg *axiom.Config) { afterCount++ },
		},
	}

	cfg := &axiom.Config{}

	h.ApplyBeforeSubTest(cfg)
	h.ApplyAfterSubTest(cfg)

	assert.Equal(t, 2, beforeCount)
	assert.Equal(t, 1, afterCount)
}

func TestHooks_Join(t *testing.T) {
	var a, b, c int

	h1 := axiom.Hooks{
		BeforeTest: []axiom.TestHook{func(cfg *axiom.Config) { a++ }},
		AfterTest:  []axiom.TestHook{func(cfg *axiom.Config) { b++ }},
	}

	h2 := axiom.Hooks{
		BeforeTest:    []axiom.TestHook{func(cfg *axiom.Config) { a += 10 }},
		AfterTest:     []axiom.TestHook{func(cfg *axiom.Config) { b += 20 }},
		BeforeSubTest: []axiom.SubTestHook{func(cfg *axiom.Config) { c++ }},
	}

	merged := h1.Join(h2)

	assert.Len(t, merged.BeforeTest, 2)
	assert.Len(t, merged.AfterTest, 2)
	assert.Len(t, merged.BeforeSubTest, 1)

	// Now simulate execution order
	cfg := &axiom.Config{}

	merged.BeforeTest[0](cfg) // a++
	merged.BeforeTest[1](cfg) // a+=10

	assert.Equal(t, 11, a)

	merged.AfterTest[0](cfg) // b++
	merged.AfterTest[1](cfg) // b+=20

	assert.Equal(t, 21, b)

	merged.BeforeSubTest[0](cfg)
	assert.Equal(t, 1, c)
}
