package axiom_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func newIntParamFixture(name string) axiom.ParamFixture[int, string] {
	return axiom.DefineParamFixture(
		name,
		func(cfg *axiom.Config, n int) (string, func(), error) {
			return fmt.Sprintf("value-%d", n), nil, nil
		},
	)
}

func TestParamFixture_ForSelectsVariantPerCase(t *testing.T) {
	pf := newIntParamFixture("atm-by-status")
	runner := axiom.NewRunner()

	var first, second string
	runner.RunCase(
		t,
		axiom.NewCase(axiom.WithCaseName("first"), axiom.WithCaseFixtures(pf.For(1))),
		func(cfg *axiom.Config) { first = pf.Get(cfg) },
	)
	runner.RunCase(
		t,
		axiom.NewCase(axiom.WithCaseName("second"), axiom.WithCaseFixtures(pf.For(2))),
		func(cfg *axiom.Config) { second = pf.Get(cfg) },
	)

	assert.Equal(t, "value-1", first)
	assert.Equal(t, "value-2", second)
	assert.Equal(t, "atm-by-status", pf.Name())
	assert.Equal(t, "atm-by-status", pf.Key().Name())
}

func TestParamFixture_DefaultUsedAndOverriddenByFor(t *testing.T) {
	pf := newIntParamFixture("with-default")
	runner := axiom.NewRunner(pf.Default(7))

	var fallback, overridden string
	// A case that does not select its own variant falls back to the default.
	runner.RunCase(
		t,
		axiom.NewCase(axiom.WithCaseName("fallback")),
		func(cfg *axiom.Config) { fallback = pf.Get(cfg) },
	)
	// A case-level For overrides the runner default (case fixtures win the merge).
	runner.RunCase(
		t,
		axiom.NewCase(axiom.WithCaseName("override"), axiom.WithCaseFixtures(pf.For(9))),
		func(cfg *axiom.Config) { overridden = pf.Get(cfg) },
	)

	assert.Equal(t, "value-7", fallback)
	assert.Equal(t, "value-9", overridden)
}

func TestWithCaseFixtures_RegistersSelfDescribingDef(t *testing.T) {
	// WithCaseFixtures must also accept an ordinary FixtureDef, proving the
	// case-level registrar is symmetric with WithRunnerFixtures.
	def := axiom.DefineFixture("case-def", func(cfg *axiom.Config) (string, func(), error) {
		return "def-value", nil, nil
	})
	runner := axiom.NewRunner()

	var got string
	runner.RunCase(
		t,
		axiom.NewCase(axiom.WithCaseName("case"), axiom.WithCaseFixtures(def)),
		func(cfg *axiom.Config) { got = def.Get(cfg) },
	)

	assert.Equal(t, "def-value", got)
}

func TestParamFixture_RunsCleanup(t *testing.T) {
	cleaned := false
	pf := axiom.DefineParamFixture(
		"with-cleanup",
		func(cfg *axiom.Config, n int) (string, func(), error) {
			return fmt.Sprintf("value-%d", n), func() { cleaned = true }, nil
		},
	)
	runner := axiom.NewRunner()

	runner.RunCase(
		t,
		axiom.NewCase(axiom.WithCaseName("case"), axiom.WithCaseFixtures(pf.For(1))),
		func(cfg *axiom.Config) { _ = pf.Get(cfg) },
	)

	assert.True(t, cleaned, "variant cleanup must run after the case finishes")
}

func TestDefineParamFixture_PanicsOnEmptyName(t *testing.T) {
	assert.PanicsWithValue(t, "fixture: key name must not be empty", func() {
		_ = axiom.DefineParamFixture("", func(*axiom.Config, int) (string, func(), error) {
			return "", nil, nil
		})
	})
}

func TestDefineParamFixture_PanicsOnNilConstructor(t *testing.T) {
	assert.PanicsWithValue(t, "fixture: nil constructor", func() {
		_ = axiom.DefineParamFixture[int, string]("x", nil)
	})
}

func TestParamFixture_ZeroValuePanics(t *testing.T) {
	const message = "fixture: param fixture must be created with DefineParamFixture"

	t.Run("for", func(t *testing.T) {
		var pf axiom.ParamFixture[int, string]
		assert.PanicsWithValue(t, message, func() { _ = pf.For(1) })
	})

	t.Run("default", func(t *testing.T) {
		var pf axiom.ParamFixture[int, string]
		assert.PanicsWithValue(t, message, func() { _ = pf.Default(1) })
	})

	t.Run("get", func(t *testing.T) {
		var pf axiom.ParamFixture[int, string]
		assert.PanicsWithValue(t, message, func() { _ = pf.Get(&axiom.Config{}) })
	})
}
