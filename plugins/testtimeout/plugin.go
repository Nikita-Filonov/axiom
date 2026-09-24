package testtimeout

import (
	"fmt"
	"runtime"
	"time"

	"github.com/Nikita-Filonov/axiom"
)

// Plugin reports test attempts that exceed the configured timeout and marks
// their test as failed. On timeout the wrapper returns, but the test body keeps
// running in its goroutine; Plugin does not cancel or stop it.
func Plugin(options ...ConfigOption) axiom.Plugin {
	cfg := NewConfig(options...)

	return func(e *axiom.Config) {
		if cfg.Timeout <= 0 {
			return
		}

		e.Runtime.EmitTestWrap(func(next axiom.TestAction) axiom.TestAction {
			return func(c *axiom.Config) {
				runWithTimeout(c, cfg, next)
			}
		})
	}
}

func runWithTimeout(c *axiom.Config, cfg Config, next axiom.TestAction) {
	done := make(chan struct{})

	var (
		panicValue    any
		panicOccurred bool
	)

	go func() {
		defer close(done)
		defer func() {
			if r := recover(); r != nil {
				panicValue, panicOccurred = r, true
			}
		}()

		next(c)
	}()

	timer := time.NewTimer(cfg.Timeout)
	defer timer.Stop()

	select {
	case <-done:
		if panicOccurred {
			panic(panicValue)
		}
	case <-timer.C:
		reportTimeout(c, cfg)
	}
}

func reportTimeout(c *axiom.Config, cfg Config) {
	if cfg.DumpGoroutines {
		c.Artefact(axiom.NewTextArtefact("testtimeout-goroutines.txt", goroutineDump()))
	}

	message := cfg.Message
	if message == "" {
		message = fmt.Sprintf("testtimeout: case exceeded %s", cfg.Timeout)
	}

	if t := c.T(); t != nil {
		t.Errorf("%s", message)
	}
}

func goroutineDump() string {
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)

	return string(buf[:n])
}
