package axiom

// SuiteTestConfig overrides the Runner or parallel policy for one registered
// suite test. An unset Runner inherits the SuiteConfig Runner.
type SuiteTestConfig struct {
	Runner   *Runner
	Parallel bool
}

// SuiteTestConfigOption configures one registered suite test.
type SuiteTestConfigOption func(*SuiteTestConfig)

// NewSuiteTestConfig returns a SuiteTestConfig with the supplied options.
func NewSuiteTestConfig(options ...SuiteTestConfigOption) SuiteTestConfig {
	cfg := SuiteTestConfig{}
	for _, option := range options {
		option(&cfg)
	}

	return cfg
}

func WithSuiteTestRunner(runner *Runner) SuiteTestConfigOption {
	return func(cfg *SuiteTestConfig) { cfg.Runner = runner }
}

// WithSuiteTestParallel runs one registered suite test in parallel. It requires
// NewSuiteFactory to provide a separate suite instance.
func WithSuiteTestParallel() SuiteTestConfigOption {
	return func(cfg *SuiteTestConfig) { cfg.Parallel = true }
}
