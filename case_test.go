package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewCase_Defaults(t *testing.T) {
	c := axiom.NewCase()

	assert.Empty(t, c.ID)
	assert.Empty(t, c.Name)
	assert.Empty(t, c.Params)
	assert.Empty(t, c.Plugins)
	assert.False(t, c.Parallel.Enabled)
	assert.Empty(t, c.Fixtures.Registry)
}

func TestWithCaseID(t *testing.T) {
	c := axiom.NewCase(axiom.WithCaseID("123"))

	assert.Equal(t, "123", c.ID)
}

func TestWithCaseName(t *testing.T) {
	c := axiom.NewCase(axiom.WithCaseName("my test"))

	assert.Equal(t, "my test", c.Name)
}

func TestWithCaseSkip(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseSkip(axiom.WithSkipReason("first")),
		axiom.WithCaseSkip(axiom.WithSkipEnabled(true)),
	)

	assert.True(t, c.Skip.Enabled)
	assert.Equal(t, "first", c.Skip.Reason)
}

func TestWithCaseMeta(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseMeta(axiom.WithMetaEpic("A")),
		axiom.WithCaseMeta(axiom.WithMetaStory("S")),
	)

	assert.Equal(t, "A", c.Meta.Epic)
	assert.Equal(t, "S", c.Meta.Story)
}

func TestWithCaseRetry(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseRetry(axiom.WithRetryTimes(5)),
		axiom.WithCaseRetry(axiom.WithRetryDelay(10)),
	)

	assert.Equal(t, 5, c.Retry.Times)
	assert.Equal(t, 10, int(c.Retry.Delay))
}

func TestWithCaseParams(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseParams(map[string]any{"u": 1}),
	)

	p := c.Params.(map[string]any)

	assert.Equal(t, 1, p["u"])
}

func TestWithCaseContext(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseContext(axiom.WithContextData("a", 1)),
		axiom.WithCaseContext(axiom.WithContextData("b", 2)),
	)

	assert.Equal(t, 1, c.Context.Data["a"])
	assert.Equal(t, 2, c.Context.Data["b"])
}

func TestWithCasePlugins(t *testing.T) {
	p1 := func(cfg *axiom.Config) {}
	p2 := func(cfg *axiom.Config) {}

	c := axiom.NewCase(
		axiom.WithCasePlugins(p1, p2),
	)

	assert.Equal(t, 2, len(c.Plugins))
}

func TestWithCaseParallel(t *testing.T) {
	c := axiom.NewCase(axiom.WithCaseParallel())

	assert.True(t, c.Parallel.Enabled)
}

func TestWithCaseSequential(t *testing.T) {
	c := axiom.NewCase(axiom.WithCaseParallel(), axiom.WithCaseSequential())

	assert.False(t, c.Parallel.Enabled)
}

func TestWithCaseFixture(t *testing.T) {
	fx := func(cfg *axiom.Config) (any, func(), error) {
		return 100, nil, nil
	}

	c := axiom.NewCase(
		axiom.WithCaseFixture("user", fx),
	)

	assert.NotNil(t, c.Fixtures.Registry)
	assert.Contains(t, c.Fixtures.Registry, "user")
}
