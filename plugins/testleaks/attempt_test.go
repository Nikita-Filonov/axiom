package testleaks

import (
	"strings"
	"testing"

	"github.com/Nikita-Filonov/axiom"
)

func TestWrapAttemptWithoutTestingT(t *testing.T) {
	called := false
	wrapped := wrapAttempt(newAttemptState(), Config{})(func(*axiom.Config) { called = true })
	wrapped(&axiom.Config{})
	if !called {
		t.Fatal("wrapped action was not called")
	}
}

func TestWrapAttemptWithoutParentContext(t *testing.T) {
	t.Run("attempt", func(t *testing.T) {
		cfg := &axiom.Config{RootT: t}
		wrapped := wrapAttempt(newAttemptState(), Config{})(func(current *axiom.Config) {
			if current != cfg {
				t.Fatal("wrapped action received a different config")
			}
		})
		wrapped(cfg)
	})
}

func TestAttemptID(t *testing.T) {
	cfg := &axiom.Config{}
	first := attemptID(cfg)
	second := attemptID(cfg)
	if first == "" || second == "" || first == second {
		t.Fatalf("random attempt IDs are not distinct: %q, %q", first, second)
	}
	cfg.Execution.ID = "execution"
	cfg.Execution.Attempt = 2
	if got := attemptID(cfg); got != "execution/2" {
		t.Fatalf("attempt ID = %q", got)
	}
	if !strings.HasPrefix(labelKey, "github.com/Nikita-Filonov/axiom/plugins/testleaks") {
		t.Fatalf("pprof label key is not namespaced: %q", labelKey)
	}
}
