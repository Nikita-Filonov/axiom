package testassert_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testassert"
)

func TestPlugin_AssertSink_NoSubT_DoesNothing(t *testing.T) {
	var called bool

	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeAssertSink(func(a axiom.Assert) { called = true }),
		),
	}

	testassert.Plugin()(cfg)

	cfg.Runtime.Assert(axiom.NewEqualAssert(1, 1, "msg"))

	if !called {
		t.Fatalf("expected assert sink to be called")
	}
}

func TestPlugin_AssertSink_EvaluatesWithSubT(t *testing.T) {
	cfg := &axiom.Config{SubT: t, Runtime: axiom.NewRuntime()}
	testassert.Plugin()(cfg)

	cfg.Runtime.Assert(axiom.NewEqualAssert(3, 3, "values must match"))
}
