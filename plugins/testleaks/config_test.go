package testleaks

import (
	"testing"
	"time"
)

func TestWithIgnoreFunction(t *testing.T) {
	c := newConfig(WithIgnoreFunction("worker"), WithGracePeriod(0), WithoutGoroutines())
	if len(c.IgnoreFunctions) != 1 || c.IgnoreFunctions[0] != "worker" || c.GracePeriod != 0 || c.Goroutines {
		t.Fatalf("config = %+v", c)
	}
	defer func() {
		if recover() == nil {
			t.Error("empty function name did not panic")
		}
	}()
	WithIgnoreFunction("")
}

func TestDefaultConfig(t *testing.T) {
	c := newConfig()
	if c.GracePeriod != 200*time.Millisecond || !c.Goroutines || len(c.IgnoreFunctions) != 0 {
		t.Fatalf("default config = %+v", c)
	}
}
