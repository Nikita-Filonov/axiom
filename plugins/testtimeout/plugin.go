package testtimeout

import "github.com/Nikita-Filonov/axiom"

// Plugin reports test attempts that exceed the configured timeout and marks
// their test as failed. On timeout the wrapper returns, but the test body keeps
// running in its goroutine. WithContextDeadline cancels its contexts, but does
// not forcibly stop the body.
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
