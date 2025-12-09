package axiom_test

import (
	"context"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewContext_Defaults(t *testing.T) {
	c := axiom.NewContext()

	assert.Nil(t, c.Raw)
	assert.Nil(t, c.GRPC)
	assert.Nil(t, c.HTTP)
	assert.Nil(t, c.Kafka)

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
	c := axiom.Context{}
	c.Normalize()

	assert.NotNil(t, c.Raw)
	assert.Equal(t, c.Raw, c.GRPC)
	assert.Equal(t, c.Raw, c.HTTP)
	assert.Equal(t, c.Raw, c.Kafka)
	assert.NotNil(t, c.Data)
}

func TestContextNormalize_DoesNotOverrideExisting(t *testing.T) {
	raw := context.WithValue(context.Background(), "k", "v")
	c := axiom.Context{
		Raw:   raw,
		GRPC:  raw,
		HTTP:  raw,
		Kafka: raw,
		Data:  map[string]any{"x": 1},
	}

	c.Normalize()

	assert.Equal(t, raw, c.Raw)
	assert.Equal(t, raw, c.GRPC)
	assert.Equal(t, raw, c.HTTP)
	assert.Equal(t, raw, c.Kafka)
	assert.Equal(t, 1, c.Data["x"])
}

func TestContextJoin_OverrideOnlyNonNil(t *testing.T) {
	baseRaw := context.WithValue(context.Background(), "a", 1)
	otherRaw := context.WithValue(context.Background(), "b", 2)

	base := axiom.Context{
		Raw:   baseRaw,
		GRPC:  baseRaw,
		HTTP:  baseRaw,
		Kafka: baseRaw,
		Data:  map[string]any{"x": 1},
	}

	other := axiom.Context{
		Raw:  otherRaw,
		HTTP: otherRaw,
		Data: map[string]any{"y": 2},
	}

	result := base.Join(other)

	// Raw overridden
	assert.Equal(t, otherRaw, result.Raw)

	// GRPC remains unchanged (other.GRPC == nil)
	assert.Equal(t, baseRaw, result.GRPC)

	// HTTP overridden
	assert.Equal(t, otherRaw, result.HTTP)

	// Kafka unchanged
	assert.Equal(t, baseRaw, result.Kafka)

	// Data merged
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
	raw := context.WithValue(context.Background(), "k", "v")

	c := axiom.NewContext(
		axiom.WithContextRaw(raw),
	)

	assert.Equal(t, raw, c.Raw)
}

func TestWithContextHTTP(t *testing.T) {
	http := context.WithValue(context.Background(), "http", 1)

	c := axiom.NewContext(
		axiom.WithContextHTTP(http),
	)

	assert.Equal(t, http, c.HTTP)
}

func TestWithContextGRPC(t *testing.T) {
	grpc := context.WithValue(context.Background(), "grpc", 2)

	c := axiom.NewContext(
		axiom.WithContextGRPC(grpc),
	)

	assert.Equal(t, grpc, c.GRPC)
}

func TestWithContextKafka(t *testing.T) {
	kafka := context.WithValue(context.Background(), "kafka", 3)

	c := axiom.NewContext(
		axiom.WithContextKafka(kafka),
	)

	assert.Equal(t, kafka, c.Kafka)
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
