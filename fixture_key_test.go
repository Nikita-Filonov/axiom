package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type dbConn struct {
	dsn string
}

func TestFixtureKey_RegisterAndGet(t *testing.T) {
	key := axiom.NewFixtureKey[*dbConn]("db")
	runner := axiom.NewRunner(
		axiom.WithRunnerFixtureKey(key, func(cfg *axiom.Config) (*dbConn, func(), error) {
			return &dbConn{dsn: cfg.Case.Name}, nil, nil
		}),
	)

	var got *dbConn
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("case")), func(cfg *axiom.Config) {
		got = key.Get(cfg)
	})

	require.NotNil(t, got)
	assert.Equal(t, "case", got.dsn)
}

func TestFixtureKey_InteropWithStringRegistry(t *testing.T) {
	// A typed key must resolve a fixture registered through the plain string API,
	// proving the layer is purely additive.
	key := axiom.NewFixtureKey[string]("greeting")
	runner := axiom.NewRunner(
		axiom.WithRunnerFixture("greeting", func(cfg *axiom.Config) (any, func(), error) {
			return "hello", nil, nil
		}),
	)

	var got string
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("case")), func(cfg *axiom.Config) {
		got = key.Get(cfg)
	})

	assert.Equal(t, "hello", got)
}

func TestFixtureKey_RunsCleanupThroughTypedRegistration(t *testing.T) {
	cleaned := false
	key := axiom.NewFixtureKey[string]("resource")
	runner := axiom.NewRunner(
		axiom.WithRunnerFixtureKey(key, func(cfg *axiom.Config) (string, func(), error) {
			return "value", func() { cleaned = true }, nil
		}),
	)

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("case")), func(cfg *axiom.Config) {
		assert.Equal(t, "value", key.Get(cfg))
	})

	assert.True(t, cleaned, "per-test fixture cleanup must run after the case finishes")
}

func TestFixtureKey_PanicsWhenNameIsEmpty(t *testing.T) {
	assert.PanicsWithValue(t, "fixture: key name must not be empty", func() {
		_ = axiom.NewFixtureKey[int]("")
	})
}

func TestFixtureKey_ZeroValueGetPanics(t *testing.T) {
	var key axiom.FixtureKey[int]

	assert.PanicsWithValue(t, "fixture: key must be created with NewFixtureKey", func() {
		_ = key.Get(&axiom.Config{})
	})
}

func TestWithRunnerFixtureKey_PanicsOnNilConstructor(t *testing.T) {
	key := axiom.NewFixtureKey[int]("x")

	assert.PanicsWithValue(t, "fixture: nil constructor", func() {
		_ = axiom.WithRunnerFixtureKey(key, nil)
	})
}

func TestWithRunnerFixtureKey_PanicsOnZeroKey(t *testing.T) {
	var key axiom.FixtureKey[int]

	assert.PanicsWithValue(t, "fixture: key must be created with NewFixtureKey", func() {
		_ = axiom.WithRunnerFixtureKey(key, func(*axiom.Config) (int, func(), error) {
			return 0, nil, nil
		})
	})
}

func TestFixtureDef_SelfRegistersAndGet(t *testing.T) {
	def := axiom.DefineFixture("db", func(cfg *axiom.Config) (*dbConn, func(), error) {
		return &dbConn{dsn: cfg.Case.Name}, nil, nil
	})
	runner := axiom.NewRunner(axiom.WithRunnerFixtures(def))

	var got *dbConn
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("case")), func(cfg *axiom.Config) {
		got = def.Get(cfg)
	})

	require.NotNil(t, got)
	assert.Equal(t, "case", got.dsn)
	assert.Equal(t, "db", def.Name())
	assert.Equal(t, "db", def.Key().Name())
}

func TestDefineFixture_PanicsOnNilConstructor(t *testing.T) {
	assert.PanicsWithValue(t, "fixture: nil constructor", func() {
		_ = axiom.DefineFixture[int]("x", nil)
	})
}
