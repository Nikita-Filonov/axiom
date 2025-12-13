package axiom_test

import (
	"context"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

type ctxKey string

func TestNewContext_Defaults(t *testing.T) {
	c := axiom.NewContext()

	assert.Nil(t, c.Raw)
	assert.Nil(t, c.DB)
	assert.Nil(t, c.MQ)
	assert.Nil(t, c.RPC)
	assert.Nil(t, c.Data)
}

func TestNewContext_WithData(t *testing.T) {
	c := axiom.NewContext(
		axiom.WithContextData("user", 123),
		axiom.WithContextData("token", "abc"),
	)

	assert.Equal(t, 123, c.Data["user"])
	assert.Equal(t, "abc", c.Data["token"])
}

func TestContextNormalize_SetsDefaults(t *testing.T) {
	var c axiom.Context
	c.Normalize()

	assert.NotNil(t, c.Raw)
	assert.Equal(t, c.Raw, c.DB)
	assert.Equal(t, c.Raw, c.MQ)
	assert.Equal(t, c.Raw, c.RPC)
	assert.NotNil(t, c.Data)
}

func TestContextNormalize_DoesNotOverrideExisting(t *testing.T) {
	raw := context.WithValue(context.Background(), ctxKey("k"), "v")
	c := axiom.Context{
		Raw:  raw,
		DB:   raw,
		MQ:   raw,
		RPC:  raw,
		Data: map[string]any{"x": 1},
	}

	c.Normalize()

	assert.Equal(t, raw, c.Raw)
	assert.Equal(t, raw, c.DB)
	assert.Equal(t, raw, c.MQ)
	assert.Equal(t, raw, c.RPC)
	assert.Equal(t, 1, c.Data["x"])
}

func TestContextJoin_OverrideOnlyNonNil(t *testing.T) {
	baseRaw := context.WithValue(context.Background(), ctxKey("a"), 1)
	otherRaw := context.WithValue(context.Background(), ctxKey("b"), 2)

	base := axiom.Context{
		Raw:  baseRaw,
		DB:   baseRaw,
		MQ:   baseRaw,
		RPC:  baseRaw,
		Data: map[string]any{"x": 1},
	}

	other := axiom.Context{
		Raw:  otherRaw,
		RPC:  otherRaw,
		Data: map[string]any{"y": 2},
	}

	result := base.Join(other)

	assert.Equal(t, otherRaw, result.Raw)
	assert.Equal(t, baseRaw, result.DB)
	assert.Equal(t, baseRaw, result.MQ)
	assert.Equal(t, otherRaw, result.RPC)

	assert.Equal(t, 1, result.Data["x"])
	assert.Equal(t, 2, result.Data["y"])
}

func TestContextJoin_DataOverwrite(t *testing.T) {
	base := axiom.Context{
		Data: map[string]any{"A": 1, "B": 2},
	}
	other := axiom.Context{
		Data: map[string]any{"B": 100, "C": 200},
	}

	result := base.Join(other)

	assert.Equal(t, 1, result.Data["A"])
	assert.Equal(t, 100, result.Data["B"]) // overwritten
	assert.Equal(t, 200, result.Data["C"])
}

func TestWithContextRaw(t *testing.T) {
	raw := context.WithValue(context.Background(), ctxKey("k"), "v")

	c := axiom.NewContext(
		axiom.WithContextRaw(raw),
	)

	assert.Equal(t, raw, c.Raw)
}

func TestWithContextDB(t *testing.T) {
	db := context.WithValue(context.Background(), ctxKey("db"), 1)
	c := axiom.NewContext(axiom.WithContextDB(db))
	assert.Equal(t, db, c.DB)
}

func TestWithContextMQ(t *testing.T) {
	mq := context.WithValue(context.Background(), ctxKey("mq"), 2)
	c := axiom.NewContext(axiom.WithContextMQ(mq))
	assert.Equal(t, mq, c.MQ)
}

func TestWithContextRPC(t *testing.T) {
	rpc := context.WithValue(context.Background(), ctxKey("rpc"), 3)
	c := axiom.NewContext(axiom.WithContextRPC(rpc))
	assert.Equal(t, rpc, c.RPC)
}

func TestGetContextValue_FoundAndTyped(t *testing.T) {
	c := axiom.NewContext(
		axiom.WithContextData("n", 42),
	)

	v, ok := axiom.GetContextValue[int](&c, "n")
	assert.True(t, ok)
	assert.Equal(t, 42, v)
}

func TestGetContextValue_NotFound(t *testing.T) {
	c := axiom.NewContext()

	v, ok := axiom.GetContextValue[string](&c, "missing")
	assert.False(t, ok)
	assert.Equal(t, "", v)
}

func TestMustContextValue_Found(t *testing.T) {
	c := axiom.NewContext(
		axiom.WithContextData("x", "hello"),
	)

	assert.Equal(t, "hello", axiom.MustContextValue[string](&c, "x"))
}

func TestMustContextValue_PanicsOnMissing(t *testing.T) {
	c := axiom.NewContext()

	assert.Panics(t, func() {
		axiom.MustContextValue[int](&c, "x")
	})
}

func TestContext_SetData(t *testing.T) {
	var c axiom.Context

	assert.Nil(t, c.Data)

	c.SetData("key1", 100)
	assert.NotNil(t, c.Data)
	assert.Equal(t, 100, c.Data["key1"])

	c.SetData("key2", "value")
	assert.Equal(t, "value", c.Data["key2"])

	c.SetData("key1", 999)
	assert.Equal(t, 999, c.Data["key1"])
}

func TestContext_SetData_ThenMustGet(t *testing.T) {
	var c axiom.Context

	c.SetData("x", 10)
	assert.Equal(t, 10, axiom.MustContextValue[int](&c, "x"))
}
