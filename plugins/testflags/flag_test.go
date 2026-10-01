package testflags_test

import (
	"flag"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testflags"
	"github.com/stretchr/testify/require"
)

func newSet(t *testing.T) *flag.FlagSet {
	t.Helper()
	fs := flag.NewFlagSet(t.Name(), flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func parse(t *testing.T, fs *flag.FlagSet, args ...string) {
	t.Helper()
	require.NoError(t, fs.Parse(args))
}

func runnerFor(fs *flag.FlagSet) *axiom.Runner {
	return axiom.NewRunner(axiom.WithRunnerResources(testflags.Resource(testflags.WithSource(fs))))
}

func checkFlag[T comparable](t *testing.T, makeKey func(...testflags.FlagOption) testflags.FlagKey[T], defaultValue T, raw string, want T) {
	t.Helper()
	for _, supplied := range []bool{false, true} {
		t.Run(fmt.Sprint(supplied), func(t *testing.T) {
			fs := newSet(t)
			key := makeKey(testflags.WithName("custom"), testflags.WithDefault(defaultValue), testflags.WithUsage("a value"), testflags.WithFlagSet(fs))
			require.Equal(t, "custom", key.Name())
			r := runnerFor(fs) // Runner construction must not capture defaults.
			expected := defaultValue
			if supplied {
				parse(t, fs, "-custom="+raw)
				expected = want
			} else {
				parse(t, fs)
			}
			require.Equal(t, expected, key.Get(r))
			entry, ok := testflags.Get(r).Lookup("custom")
			require.True(t, ok)
			require.Equal(t, testflags.Flag{
				Set: supplied, Name: "custom", Value: expected,
				Usage: "a value", Default: fmt.Sprint(defaultValue),
			}, entry)
		})
	}
}

func TestTypedFlags(t *testing.T) {
	t.Run("bool", func(t *testing.T) { checkFlag(t, testflags.Bool, true, "false", false) })
	t.Run("int", func(t *testing.T) { checkFlag(t, testflags.Int, 7, "-55", -55) })
	t.Run("int64", func(t *testing.T) {
		checkFlag(t, testflags.Int64, int64(7), "9223372036854775807", int64(9223372036854775807))
	})
	t.Run("uint", func(t *testing.T) { checkFlag(t, testflags.Uint, uint(7), "0", uint(0)) })
	t.Run("uint64", func(t *testing.T) {
		checkFlag(t, testflags.Uint64, uint64(7), "18446744073709551615", uint64(18446744073709551615))
	})
	t.Run("string", func(t *testing.T) { checkFlag(t, testflags.String, "local", "", "") })
	t.Run("float64", func(t *testing.T) { checkFlag(t, testflags.Float64, 1.5, "-0.25", -0.25) })
	t.Run("duration", func(t *testing.T) { checkFlag(t, testflags.Duration, time.Second, "250ms", 250*time.Millisecond) })
}

func TestFlagOptionsAndZeroDefaults(t *testing.T) {
	fs := newSet(t)
	key := testflags.Int(testflags.WithName("old"), testflags.WithName("custom"), testflags.WithDefault(1), testflags.WithDefault(2), testflags.WithFlagSet(fs))
	boolean := testflags.Bool(testflags.WithName("enabled"), testflags.WithFlagSet(fs))
	text := testflags.String(testflags.WithName("text"), testflags.WithFlagSet(fs))
	zero := testflags.Int(testflags.WithName("zero"), testflags.WithFlagSet(fs))
	parse(t, fs, "-enabled", "-custom=0", "-custom=55", "--", "-zero=99")
	r := runnerFor(fs)
	require.Equal(t, 55, key.Get(r))
	require.True(t, boolean.Get(r))
	require.Empty(t, text.Get(r))
	require.Equal(t, 0, zero.Get(r))
	require.Nil(t, fs.Lookup("old"))
}

func TestFlagDeclarationValidation(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
		opts []testflags.FlagOption
	}{
		{"missing name", "testflags: flag name must not be empty", nil},
		{"nil option", "testflags: nil flag option", []testflags.FlagOption{nil}},
		{"nil set", "testflags: nil flag set", []testflags.FlagOption{testflags.WithName("x"), testflags.WithFlagSet(nil)}},
		{"wrong default", `testflags: default for "x" must have type int, got string`, []testflags.FlagOption{testflags.WithName("x"), testflags.WithDefault("1")}},
		{"leading dash", `flag "-x" begins with -`, []testflags.FlagOption{testflags.WithName("-x")}},
		{"equals", `flag "x=y" contains =`, []testflags.FlagOption{testflags.WithName("x=y")}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			opts := append([]testflags.FlagOption{testflags.WithFlagSet(newSet(t))}, tc.opts...)
			require.PanicsWithValue(t, tc.want, func() { testflags.Int(opts...) })
		})
	}
	require.PanicsWithValue(t, "testflags: nil default", func() { testflags.WithDefault(nil) })
	fs := newSet(t)
	opts := []testflags.FlagOption{testflags.WithName("x"), testflags.WithFlagSet(fs)}
	testflags.Int(opts...)
	require.PanicsWithValue(t, fs.Name()+" flag redefined: x", func() { testflags.Int(opts...) })
	parse(t, fs)
	require.PanicsWithValue(t, "testflags: flags must be registered before parsing", func() { testflags.Int(opts...) })
	require.PanicsWithValue(t, `testflags: default for "x" must have type time.Duration, got int`, func() {
		testflags.Duration(testflags.WithName("x"), testflags.WithDefault(1), testflags.WithFlagSet(newSet(t)))
	})
}

func TestParserRejectsInvalidValues(t *testing.T) {
	for _, tc := range []struct {
		name string
		add  func(*flag.FlagSet)
		arg  string
	}{
		{"int", func(fs *flag.FlagSet) { testflags.Int(testflags.WithName("x"), testflags.WithFlagSet(fs)) }, "text"},
		{"overflow", func(fs *flag.FlagSet) { testflags.Int64(testflags.WithName("x"), testflags.WithFlagSet(fs)) }, "9223372036854775808"},
		{"negative unsigned", func(fs *flag.FlagSet) { testflags.Uint(testflags.WithName("x"), testflags.WithFlagSet(fs)) }, "-1"},
		{"bool", func(fs *flag.FlagSet) { testflags.Bool(testflags.WithName("x"), testflags.WithFlagSet(fs)) }, "yes"},
		{"duration", func(fs *flag.FlagSet) { testflags.Duration(testflags.WithName("x"), testflags.WithFlagSet(fs)) }, "tomorrow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			fs := newSet(t)
			tc.add(fs)
			require.Error(t, fs.Parse([]string{"-x=" + tc.arg}))
		})
	}
}

func TestTypedKeyValidationAndInterop(t *testing.T) {
	fs := newSet(t)
	fs.Int("existing", 42, "declared outside testflags")
	parse(t, fs)
	r := runnerFor(fs)
	require.Equal(t, 42, testflags.NewFlagKey[int]("existing").Get(r))
	for _, tc := range []struct {
		key  testflags.FlagKey[string]
		want string
	}{
		{testflags.NewFlagKey[string]("existing"), `testflags: flag "existing" has type int, expected string`},
		{testflags.NewFlagKey[string]("absent"), `testflags: flag "absent" not found`},
	} {
		value, err := tc.key.TryGet(r)
		require.Empty(t, value)
		require.EqualError(t, err, tc.want)
		require.PanicsWithError(t, tc.want, func() { tc.key.Get(r) })
	}
	require.PanicsWithValue(t, "testflags: flag name must not be empty", func() { testflags.NewFlagKey[int]("") })
	require.PanicsWithValue(t, "testflags: key must be created with NewFlagKey or a flag constructor", func() {
		var key testflags.FlagKey[int]
		key.Get(r)
	})
	key := testflags.NewFlagKey[int]("existing")
	v, err := key.TryGet(nil)
	require.Zero(t, v)
	require.EqualError(t, err, "testflags: nil runner")
	_, err = key.TryGet(axiom.NewRunner())
	require.ErrorContains(t, err, "not found")
}
