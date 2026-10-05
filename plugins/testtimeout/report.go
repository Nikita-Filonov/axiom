package testtimeout

import (
	"fmt"
	"runtime"

	"github.com/Nikita-Filonov/axiom"
)

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
