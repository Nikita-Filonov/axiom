package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type localBundle struct {
	Name string
}

type otherLocalBundle struct {
	ID int
}

func TestLocal_SetGetAndOverwrite(t *testing.T) {
	cfg := &axiom.Config{}
	key := axiom.NewLocalKey[localBundle]("bundle")

	_, ok := axiom.GetLocal(cfg, key)
	assert.False(t, ok)

	axiom.SetLocal(cfg, key, localBundle{Name: "first"})

	got, ok := axiom.GetLocal(cfg, key)
	require.True(t, ok)
	assert.Equal(t, localBundle{Name: "first"}, got)

	axiom.SetLocal(cfg, key, localBundle{Name: "second"})

	got, ok = axiom.GetLocal(cfg, key)
	require.True(t, ok)
	assert.Equal(t, localBundle{Name: "second"}, got)
}

func TestLocal_UsesNameAndTypeAsKey(t *testing.T) {
	cfg := &axiom.Config{}
	first := axiom.NewLocalKey[localBundle]("same")
	second := axiom.NewLocalKey[localBundle]("same")

	axiom.SetLocal(cfg, first, localBundle{Name: "one"})
	axiom.SetLocal(cfg, second, localBundle{Name: "two"})

	assert.Equal(t, localBundle{Name: "two"}, axiom.MustLocal(cfg, first))
	assert.Equal(t, localBundle{Name: "two"}, axiom.MustLocal(cfg, second))
}

func TestLocal_SeparatesValuesByType(t *testing.T) {
	cfg := &axiom.Config{}
	toolsKey := axiom.NewLocalKey[localBundle]("same")
	otherKey := axiom.NewLocalKey[otherLocalBundle]("same")

	axiom.SetLocal(cfg, toolsKey, localBundle{Name: "tools"})
	axiom.SetLocal(cfg, otherKey, otherLocalBundle{ID: 42})

	tools := axiom.MustLocal(cfg, toolsKey)
	other := axiom.MustLocal(cfg, otherKey)

	assert.Equal(t, localBundle{Name: "tools"}, tools)
	assert.Equal(t, otherLocalBundle{ID: 42}, other)
}

func TestLocal_SupportsPointerValues(t *testing.T) {
	cfg := &axiom.Config{}
	key := axiom.NewLocalKey[*localBundle]("pointer")
	value := &localBundle{Name: "pointer"}

	axiom.SetLocal(cfg, key, value)

	got := axiom.MustLocal(cfg, key)
	assert.Same(t, value, got)

	got.Name = "mutated"
	assert.Equal(t, "mutated", value.Name)
}

func TestLocal_SupportsTypedNilPointers(t *testing.T) {
	cfg := &axiom.Config{}
	key := axiom.NewLocalKey[*localBundle]("pointer")
	var value *localBundle

	axiom.SetLocal(cfg, key, value)

	got, ok := axiom.GetLocal(cfg, key)
	require.True(t, ok)
	assert.Nil(t, got)
}

func TestLocal_SupportsNilInterfaceValues(t *testing.T) {
	cfg := &axiom.Config{}
	key := axiom.NewLocalKey[any]("interface")

	axiom.SetLocal[any](cfg, key, nil)

	got, ok := axiom.GetLocal(cfg, key)
	require.True(t, ok)
	assert.Nil(t, got)
	assert.Nil(t, axiom.MustLocal(cfg, key))
}

func TestLocal_MustPanicsWhenValueIsMissing(t *testing.T) {
	cfg := &axiom.Config{}
	key := axiom.NewLocalKey[localBundle]("bundle")

	assert.PanicsWithValue(t, "local: missing value for key \"bundle\"", func() {
		axiom.MustLocal(cfg, key)
	})
}

func TestLocal_PanicsOnNilConfig(t *testing.T) {
	key := axiom.NewLocalKey[localBundle]("bundle")

	assert.PanicsWithValue(t, "local: nil *Config", func() {
		axiom.SetLocal(nil, key, localBundle{})
	})
	assert.PanicsWithValue(t, "local: nil *Config", func() {
		_, _ = axiom.GetLocal(nil, key)
	})
	assert.PanicsWithValue(t, "local: nil *Config", func() {
		_ = axiom.MustLocal(nil, key)
	})
}

func TestLocalKey_PanicsWhenNameIsEmpty(t *testing.T) {
	assert.PanicsWithValue(t, "local: key name must not be empty", func() {
		_ = axiom.NewLocalKey[localBundle]("")
	})

	var key axiom.LocalKey[localBundle]
	cfg := &axiom.Config{}

	assert.PanicsWithValue(t, "local: key must be created with NewLocalKey", func() {
		axiom.SetLocal(cfg, key, localBundle{})
	})
	assert.PanicsWithValue(t, "local: key must be created with NewLocalKey", func() {
		_, _ = axiom.GetLocal(cfg, key)
	})
	assert.PanicsWithValue(t, "local: key must be created with NewLocalKey", func() {
		_ = axiom.MustLocal(cfg, key)
	})
}
