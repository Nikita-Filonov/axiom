package testenv_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testenv"
	"github.com/stretchr/testify/require"
)

func runnerWith(entries ...string) *axiom.Runner {
	return axiom.NewRunner(axiom.WithRunnerResources(testenv.Resource(testenv.WithSource(func() []string {
		return entries
	}))))
}

func checkKey[T comparable](t *testing.T, constructor func(...testenv.EnvOption) testenv.EnvKey[T], raw string, want T) {
	t.Helper()
	key := constructor(testenv.WithName("CUSTOM"))
	require.Equal(t, "CUSTOM", key.Name())
	require.Equal(t, want, key.Get(runnerWith("CUSTOM="+raw)))
	value, err := key.TryGet(runnerWith())
	require.ErrorContains(t, err, `required variable "CUSTOM" is not set`)
	require.Empty(t, value)
}

func TestTypedEnvironmentKeys(t *testing.T) {
	t.Run("string", func(t *testing.T) { checkKey(t, testenv.String, "hello=world", "hello=world") })
	t.Run("bool", func(t *testing.T) { checkKey(t, testenv.Bool, "true", true) })
	t.Run("int", func(t *testing.T) { checkKey(t, testenv.Int, "-55", -55) })
	t.Run("int64", func(t *testing.T) { checkKey(t, testenv.Int64, "9223372036854775807", int64(9223372036854775807)) })
	t.Run("uint", func(t *testing.T) { checkKey(t, testenv.Uint, "55", uint(55)) })
	t.Run("uint64", func(t *testing.T) { checkKey(t, testenv.Uint64, "18446744073709551615", uint64(18446744073709551615)) })
	t.Run("float64", func(t *testing.T) { checkKey(t, testenv.Float64, "1.5", 1.5) })
	t.Run("duration", func(t *testing.T) { checkKey(t, testenv.Duration, "250ms", 250*time.Millisecond) })
}

func TestDefaultsAndExplicitEmpty(t *testing.T) {
	key := testenv.String(testenv.WithName("CUSTOM"), testenv.WithDefault("fallback"))
	require.Equal(t, "fallback", key.Get(runnerWith()))
	require.Equal(t, "", key.Get(runnerWith("CUSTOM=")))
	require.Equal(t, false, testenv.Bool(testenv.WithName("BOOL"), testenv.WithDefault(false)).Get(runnerWith()))
	require.False(t, testenv.Bool(testenv.WithName("BOOL"), testenv.WithDefault(true)).Get(runnerWith("BOOL=false")))
	require.Equal(t, 7, testenv.Int(testenv.WithName("INT"), testenv.WithDefault(7)).Get(runnerWith()))
	require.Zero(t, testenv.Int(testenv.WithName("INT"), testenv.WithDefault(7)).Get(runnerWith("INT=0")))
	require.Equal(t, 9, testenv.Int(testenv.WithName("INT"), testenv.WithDefault(1), testenv.WithDefault(9)).Get(runnerWith()))
	_, err := testenv.Bool(testenv.WithName("BOOL"), testenv.WithDefault(true)).TryGet(runnerWith("BOOL="))
	require.ErrorContains(t, err, `testenv: invalid value for "BOOL"`)
}

func TestKeyErrors(t *testing.T) {
	key := testenv.String(testenv.WithName("REQUIRED"))
	unregistered := axiom.NewRunner()
	value, err := key.TryGet(unregistered)
	require.Empty(t, value)
	require.ErrorContains(t, err, "not found")
	require.PanicsWithError(t, err.Error(), func() { key.Get(unregistered) })
	require.PanicsWithError(t, `testenv: required variable "REQUIRED" is not set`, func() {
		key.Get(runnerWith())
	})
}

func TestMalformedValuesAndDeclarations(t *testing.T) {
	for _, tc := range []struct {
		name string
		key  func() error
	}{
		{"bool", func() error { _, err := testenv.Bool(testenv.WithName("X")).TryGet(runnerWith("X=maybe")); return err }},
		{"int", func() error { _, err := testenv.Int(testenv.WithName("X")).TryGet(runnerWith("X=")); return err }},
		{"overflow", func() error {
			_, err := testenv.Int64(testenv.WithName("X")).TryGet(runnerWith("X=9223372036854775808"))
			return err
		}},
		{"unsigned", func() error { _, err := testenv.Uint(testenv.WithName("X")).TryGet(runnerWith("X=-1")); return err }},
		{"uint64 overflow", func() error {
			_, err := testenv.Uint64(testenv.WithName("X")).TryGet(runnerWith("X=18446744073709551616"))
			return err
		}},
		{"float64", func() error {
			_, err := testenv.Float64(testenv.WithName("X")).TryGet(runnerWith("X=1.2.3"))
			return err
		}},
		{"duration", func() error {
			_, err := testenv.Duration(testenv.WithName("X")).TryGet(runnerWith("X=tomorrow"))
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.ErrorContains(t, tc.key(), `testenv: invalid value for "X"`)
		})
	}
	secret := "do-not-print-this-value"
	_, err := testenv.Int(testenv.WithName("SECRET")).TryGet(runnerWith("SECRET=" + secret))
	require.Error(t, err)
	require.NotContains(t, err.Error(), secret)
	for _, name := range []string{"", "A=B", "A\x00B"} {
		t.Run(fmt.Sprintf("invalid name %q", name), func(t *testing.T) {
			require.PanicsWithValue(t, fmt.Sprintf("testenv: invalid environment variable name %q", name), func() {
				testenv.String(testenv.WithName(name))
			})
		})
	}
	require.PanicsWithValue(t, "testenv: nil env option", func() { testenv.String(nil) })
	require.PanicsWithValue(t, "testenv: nil default", func() { testenv.WithDefault(nil) })
	require.PanicsWithValue(t, `testenv: default for "X" must have type int, got string`, func() {
		testenv.Int(testenv.WithName("X"), testenv.WithDefault("1"))
	})
	var zero testenv.EnvKey[string]
	require.PanicsWithValue(t, "testenv: key must be created with a typed constructor", func() { zero.Get(runnerWith()) })
}
