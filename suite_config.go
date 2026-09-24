package axiom

// SuiteConfig selects the Runner and parallel policy for registered suite
// tests. A missing Runner is replaced with a new one by NewSuiteConfig.
type SuiteConfig struct {
	Runner   *Runner
	Parallel bool
}

// SuiteConfigOption configures a SuiteRunner at construction.
type SuiteConfigOption func(*SuiteConfig)

// NewSuiteConfig returns a SuiteConfig, creating a Runner if none was supplied.
func NewSuiteConfig(options ...SuiteConfigOption) SuiteConfig {
	cfg := SuiteConfig{}
	for _, option := range options {
		option(&cfg)
	}

	if cfg.Runner == nil {
		cfg.Runner = NewRunner()
	}

	return cfg
}

func WithSuiteConfigRunner(runner *Runner) SuiteConfigOption {
	return func(cfg *SuiteConfig) { cfg.Runner = runner }
}

// WithSuiteConfigParallel runs registered suite tests in parallel. It requires
// NewSuiteFactory because each test needs a separate suite instance.
func WithSuiteConfigParallel() SuiteConfigOption {
	return func(cfg *SuiteConfig) { cfg.Parallel = true }
}
