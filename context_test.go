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
