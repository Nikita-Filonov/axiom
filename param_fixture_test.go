package axiom_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	runner := axiom.NewRunner(axiom.WithRunnerFixtures(pf.Default(7)))

	for _, test := range []struct {
		c    axiom.Case
		want string
	}{
		{axiom.NewCase(axiom.WithCaseName("fallback")), "value-7"},
		{axiom.NewCase(axiom.WithCaseName("override"), axiom.WithCaseFixtures(pf.For(9))), "value-9"},
		{axiom.NewCase(axiom.WithCaseName("zero override"), axiom.WithCaseFixtures(pf.For(0))), "value-0"},
		{axiom.NewCase(axiom.WithCaseName("fallback after overrides")), "value-7"},
	} {
		runner.RunCase(t, test.c, func(cfg *axiom.Config) {
			assert.Equal(cfg.T(), test.want, pf.Get(cfg))
		})
	}
}

func TestParamFixture_DefaultBindingsComposeWithOrdinaryFixtures(t *testing.T) {
	pf := newIntParamFixture("value")
	first, second := pf.Default(3), pf.Default(5)
	ordinary := axiom.DefineFixture("ordinary", func(*axiom.Config) (string, func(), error) {
		return "ordinary", nil, nil
	})
	replacement := axiom.DefineFixture(pf.Name(), func(*axiom.Config) (string, func(), error) {
		return "replacement", nil, nil
	})

	for _, test := range []struct {
		name     string
		fixtures []axiom.RunnerFixtureRegistrar
		want     string
	}{
		{"first binding stays independent", []axiom.RunnerFixtureRegistrar{first}, "value-3"},
		{"second binding", []axiom.RunnerFixtureRegistrar{second}, "value-5"},
		{"zero default", []axiom.RunnerFixtureRegistrar{pf.Default(0)}, "value-0"},
		{"last default wins", []axiom.RunnerFixtureRegistrar{first, second}, "value-5"},
		{"ordinary fixture overrides default", []axiom.RunnerFixtureRegistrar{first, replacement}, "replacement"},
		{"default overrides ordinary fixture", []axiom.RunnerFixtureRegistrar{replacement, first}, "value-3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixtures := append([]axiom.RunnerFixtureRegistrar{ordinary}, test.fixtures...)
			runner := axiom.NewRunner(axiom.WithRunnerFixtures(fixtures...))
			runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("resolve fixtures")), func(cfg *axiom.Config) {
				assert.Equal(cfg.T(), "ordinary", ordinary.Get(cfg))
				assert.Equal(cfg.T(), test.want, pf.Get(cfg))
			})
		})
	}
}

func TestParamFixture_DefaultKeepsPerAttemptLifecycle(t *testing.T) {
	type value struct {
		param int
		name  string
	}
	builds := 0
	var cleaned []string
	pf := axiom.DefineParamFixture("lifecycle", func(cfg *axiom.Config, param int) (*value, func(), error) {
		builds++
		return &value{param: param, name: cfg.Case.Name}, func() {
			cleaned = append(cleaned, cfg.Case.Name)
		}, nil
	})
	runner := axiom.NewRunner(axiom.WithRunnerFixtures(pf.Default(42)))
	require.Zero(t, builds, "registration must not construct the fixture")

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("unused")), func(*axiom.Config) {})
	require.Zero(t, builds, "unused defaults must stay lazy")
	require.Empty(t, cleaned)

	var previous *value
	names := []string{"first", "second"}
	for i, name := range names {
		runner.RunCase(t, axiom.NewCase(axiom.WithCaseName(name)), func(cfg *axiom.Config) {
			got := pf.Get(cfg)
			require.NotNil(cfg.T(), got)
			assert.Equal(cfg.T(), 42, got.param)
			assert.Equal(cfg.T(), name, got.name)
			assert.NotSame(cfg.T(), previous, got, "each attempt gets a fresh fixture")
			assert.Same(cfg.T(), got, pf.Get(cfg))
			assert.Same(cfg.T(), got, pf.Key().Get(cfg))
			assert.Same(cfg.T(), got, axiom.GetFixture[*value](cfg, pf.Name()))
			assert.Equal(cfg.T(), i+1, builds, "all accessors share the attempt cache")
			assert.Len(cfg.T(), cleaned, i, "cleanup must wait until the action finishes")
			previous = got
		})
		require.Equal(t, names[:i+1], cleaned, "each constructed fixture is cleaned up once")
	}
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
