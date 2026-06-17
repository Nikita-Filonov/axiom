package axiom

type SuiteTestConfig struct {
	Runner   *Runner
	Parallel bool
}

type SuiteTestConfigOption func(*SuiteTestConfig)

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

func WithSuiteTestParallel() SuiteTestConfigOption {
	return func(cfg *SuiteTestConfig) { cfg.Parallel = true }
}
