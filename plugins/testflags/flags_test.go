package testflags_test

import (
	"flag"
	"testing"

	"github.com/Nikita-Filonov/axiom/plugins/testflags"
	"github.com/stretchr/testify/require"
)

type textValue string

func (v *textValue) String() string     { return string(*v) }
func (v *textValue) Set(s string) error { *v = textValue(s); return nil }

func TestSnapshotMetadataAndIsolation(t *testing.T) {
	fs := newSet(t)
	fs.Int("zero", 0, "zero flag")
	fs.Bool("false", false, "")
	fs.String("empty", "", "")
	fs.String("default", "local", "environment")
	var custom textValue = "first"
	fs.Var(&custom, "text", "custom flag.Value")
	parse(t, fs, "-zero=0", "-false=false", "-empty=", "-text=second")
	require.NoError(t, fs.Set("default", "local"))
	r := runnerFor(fs)
	snapshot := testflags.Get(r)
	for _, name := range []string{"zero", "false", "empty", "default", "text"} {
		entry, ok := snapshot.Lookup(name)
		require.True(t, ok)
		require.True(t, entry.Set)
	}
	require.Equal(t, "second", testflags.NewFlagKey[string]("text").Get(r))
	value, ok := snapshot.Get("missing")
	require.Nil(t, value)
	require.False(t, ok)
	_, ok = snapshot.Lookup("missing")
	require.False(t, ok)
	all := snapshot.All()
	var names []string
	for _, entry := range all {
		names = append(names, entry.Name)
	}
	require.Equal(t, []string{"default", "empty", "false", "text", "zero"}, names)
	all[0].Name = "changed"
	all[0].Value = "changed"
	entry, _ := snapshot.Lookup("default")
	entry.Usage = "changed"
	require.NoError(t, fs.Set("default", "later"))
	require.NoError(t, fs.Set("text", "later"))
	again, _ := snapshot.Lookup("default")
	require.Equal(t, "local", again.Value)
	require.Equal(t, "default", again.Name)
	require.Equal(t, "environment", again.Usage)
	require.Equal(t, "local", again.Default)
	got, _ := snapshot.Get("text")
	require.Equal(t, "second", got)
	next, _ := testflags.Get(runnerFor(fs)).Get("default")
	require.Equal(t, "later", next)
}

func TestEmptySnapshotAndZeroValue(t *testing.T) {
	fs := newSet(t)
	parse(t, fs)
	for _, snapshot := range []*testflags.Flags{{}, testflags.Get(runnerFor(fs))} {
		require.Empty(t, snapshot.All())
		value, ok := snapshot.Get("x")
		require.Nil(t, value)
		require.False(t, ok)
		_, ok = snapshot.Lookup("x")
		require.False(t, ok)
	}
}

func TestStandardFuncFlagUsesTextFallback(t *testing.T) {
	fs := newSet(t)
	called := ""
	fs.Func("action", "run action", func(s string) error { called = s; return nil })
	parse(t, fs, "-action=value")
	snapshot := testflags.Get(runnerFor(fs))
	entry, ok := snapshot.Lookup("action")
	require.True(t, ok)
	require.True(t, entry.Set)
	require.Equal(t, "value", called)
	require.Equal(t, fs.Lookup("action").Value.String(), entry.Value)
}

// Custom Getter values keep their dynamic type, including typed nil values.
type nilGetter struct{ flag.Value }

func (*nilGetter) String() string { return "nil" }
func (*nilGetter) Get() any       { return (*int)(nil) }

func TestCustomGetterPreservesTypedNil(t *testing.T) {
	fs := newSet(t)
	fs.Var(&nilGetter{}, "optional", "")
	parse(t, fs)
	require.Nil(t, testflags.NewFlagKey[*int]("optional").Get(runnerFor(fs)))
}

type panicGetter struct {
	textValue
	calls int
}

func (v *panicGetter) Get() any {
	v.calls++
	panic("custom getter failed")
}

func TestSnapshotPanicIsRepeatedWithoutReturningPartialValues(t *testing.T) {
	fs := newSet(t)
	fs.Int("first", 1, "")
	value := &panicGetter{}
	fs.Var(value, "last", "")
	parse(t, fs)
	r := runnerFor(fs)
	require.PanicsWithValue(t, "custom getter failed", func() { testflags.Get(r) })
	require.PanicsWithValue(t, "custom getter failed", func() { _, _ = testflags.TryGet(r) })
	require.Equal(t, 1, value.calls)
}
