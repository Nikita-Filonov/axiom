package testassert_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testassert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_AssertSink_NoSubT_DoesNothing(t *testing.T) {
	var called bool

	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeAssertSink(func(a axiom.Assert) { called = true }),
		),
	}

	testassert.Plugin()(cfg)

	cfg.Assert(axiom.NewEqualAssert(1, 1, "msg"))

	require.True(t, called)
}

func TestPlugin_AssertSink_EvaluatesWithSubT(t *testing.T) {
	cfg := &axiom.Config{SubT: t, Runtime: axiom.NewRuntime()}
	testassert.Plugin()(cfg)

	cfg.Assert(axiom.NewEqualAssert(3, 3, "values must match"))
}
