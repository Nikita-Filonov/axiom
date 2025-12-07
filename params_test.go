package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

type sampleParams struct {
	Foo string
	Bar int
}

func TestGetParams_ValueSuccess(t *testing.T) {
	cfg := &axiom.Config{
		Params: sampleParams{Foo: "hello", Bar: 123},
	}

	p := axiom.GetParams[sampleParams](cfg)

	assert.Equal(t, "hello", p.Foo)
	assert.Equal(t, 123, p.Bar)
}

func TestGetParams_PointerSuccess(t *testing.T) {
	cfg := &axiom.Config{
		Params: &sampleParams{Foo: "hi", Bar: 999},
	}

	p := axiom.GetParams[*sampleParams](cfg)

	assert.Equal(t, "hi", p.Foo)
	assert.Equal(t, 999, p.Bar)
}

func TestGetParams_Panic_WrongType(t *testing.T) {
	cfg := &axiom.Config{
		Params: "not the right type",
	}

	assert.Panics(t, func() {
		_ = axiom.GetParams[sampleParams](cfg)
	})
}

func TestGetParams_Panic_Nil(t *testing.T) {
	cfg := &axiom.Config{
		Params: nil,
	}

	assert.Panics(t, func() {
		_ = axiom.GetParams[sampleParams](cfg)
	})
}

func TestGetParams_Panic_ValueProvidedButPointerExpected(t *testing.T) {
	cfg := &axiom.Config{
		Params: sampleParams{Foo: "x"},
	}

	assert.Panics(t, func() {
		_ = axiom.GetParams[*sampleParams](cfg) // expecting *sampleParams
	})
}

func TestGetParams_Panic_PointerProvidedButValueExpected(t *testing.T) {
	cfg := &axiom.Config{
		Params: &sampleParams{Foo: "x"},
	}

	assert.Panics(t, func() {
		_ = axiom.GetParams[sampleParams](cfg) // expecting value, but got pointer
	})
}
