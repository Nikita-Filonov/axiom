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
