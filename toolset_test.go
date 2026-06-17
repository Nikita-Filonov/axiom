package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type runnerTools struct {
	CaseName string
	T        *testing.T
}

func TestToolset_BindsToolsFromBeforeTestHook(t *testing.T) {
	var seen []string
	var tools []*runnerTools

	toolset := axiom.NewToolset("runner-tools", func(cfg *axiom.Config) *runnerTools {
		return &runnerTools{
			CaseName: cfg.Case.Name,
			T:        cfg.T(),
		}
	})

	runner := axiom.NewRunner(
		axiom.WithRunnerHooks(
			axiom.WithBeforeTest(toolset.Bind),
		),
	)

	run := func(name string) {
		runner.RunCase(t, axiom.NewCase(axiom.WithCaseName(name)), toolset.Use(
			func(cfg *axiom.ConfigWithTools[*runnerTools]) {
				seen = append(seen, cfg.Tools.CaseName)
				tools = append(tools, cfg.Tools)

				assert.Equal(cfg.SubT, cfg.Case.Name, cfg.Tools.CaseName)
				assert.Same(cfg.SubT, cfg.SubT, cfg.Tools.T)
			},
		))
	}

	run("first")
	run("second")

	require.Len(t, tools, 2)
	assert.Equal(t, []string{"first", "second"}, seen)
	assert.NotSame(t, tools[0], tools[1])
}

func TestToolset_PassesOriginalConfig(t *testing.T) {
	cfg := &axiom.Config{
		Case: &axiom.Case{Name: "case"},
	}
	toolset := axiom.NewToolset("tools", func(cfg *axiom.Config) localBundle {
		return localBundle{Name: cfg.Case.Name}
	})

	toolset.Bind(cfg)

	action := toolset.Use(func(cfg *axiom.ConfigWithTools[localBundle]) {
		assert.Equal(t, "case", cfg.Case.Name)
		assert.Equal(t, localBundle{Name: "case"}, cfg.Tools)
	})

	action(cfg)
}

func TestToolset_GetAndMust(t *testing.T) {
	cfg := &axiom.Config{}
	toolset := axiom.NewToolset("tools", func(cfg *axiom.Config) localBundle {
		return localBundle{Name: "tools"}
	})

	_, ok := toolset.Get(cfg)
	assert.False(t, ok)

	toolset.Bind(cfg)

	got, ok := toolset.Get(cfg)
	require.True(t, ok)
	assert.Equal(t, localBundle{Name: "tools"}, got)
	assert.Equal(t, localBundle{Name: "tools"}, toolset.Must(cfg))
}

func TestToolset_UsesNameAndTypeAsLocalKey(t *testing.T) {
	cfg := &axiom.Config{}
	first := axiom.NewToolset("same", func(cfg *axiom.Config) localBundle {
		return localBundle{Name: "first"}
	})
	second := axiom.NewToolset("same", func(cfg *axiom.Config) localBundle {
		return localBundle{Name: "second"}
	})
	other := axiom.NewToolset("same", func(cfg *axiom.Config) otherLocalBundle {
		return otherLocalBundle{ID: 42}
	})

	first.Bind(cfg)
	second.Bind(cfg)
	other.Bind(cfg)

	assert.Equal(t, localBundle{Name: "second"}, first.Must(cfg))
	assert.Equal(t, localBundle{Name: "second"}, second.Must(cfg))
	assert.Equal(t, otherLocalBundle{ID: 42}, other.Must(cfg))
}

func TestToolset_PanicsWhenToolsAreMissing(t *testing.T) {
	toolset := axiom.NewToolset("tools", func(cfg *axiom.Config) localBundle {
		return localBundle{Name: "tools"}
	})
	action := toolset.Use(func(cfg *axiom.ConfigWithTools[localBundle]) {})

	assert.PanicsWithValue(t, "local: missing value for key \"tools\"", func() {
		action(&axiom.Config{})
	})
}

func TestToolset_PanicsWhenBuildIsNil(t *testing.T) {
	assert.PanicsWithValue(t, "toolset: nil build", func() {
		_ = axiom.NewToolset[localBundle]("tools", nil)
	})
}

func TestToolset_PanicsWhenNameIsEmpty(t *testing.T) {
	assert.PanicsWithValue(t, "local: key name must not be empty", func() {
		_ = axiom.NewToolset("",
			func(cfg *axiom.Config) localBundle {
				return localBundle{}
			},
		)
	})
}

func TestToolset_PanicsWhenConfigIsNil(t *testing.T) {
	toolset := axiom.NewToolset("tools", func(cfg *axiom.Config) localBundle {
		return localBundle{}
	})

	assert.PanicsWithValue(t, "local: nil *Config", func() {
		toolset.Bind(nil)
	})
	assert.PanicsWithValue(t, "local: nil *Config", func() {
		_, _ = toolset.Get(nil)
	})
	assert.PanicsWithValue(t, "local: nil *Config", func() {
		_ = toolset.Must(nil)
	})
}

func TestToolset_PanicsWhenActionIsNil(t *testing.T) {
	toolset := axiom.NewToolset("tools", func(cfg *axiom.Config) localBundle {
		return localBundle{}
	})

	assert.PanicsWithValue(t, "toolset: nil action", func() {
		_ = toolset.Use(nil)
	})
}

func TestToolset_PanicsWhenEmpty(t *testing.T) {
	var toolset axiom.Toolset[localBundle]

	assert.PanicsWithValue(t, "toolset: empty toolset", func() {
		toolset.Bind(&axiom.Config{})
	})
	assert.PanicsWithValue(t, "toolset: empty toolset", func() {
		_ = toolset.Use(func(cfg *axiom.ConfigWithTools[localBundle]) {})
	})
	assert.PanicsWithValue(t, "toolset: empty toolset", func() {
		_, _ = toolset.Get(&axiom.Config{})
	})
	assert.PanicsWithValue(t, "toolset: empty toolset", func() {
		_ = toolset.Must(&axiom.Config{})
	})
}
