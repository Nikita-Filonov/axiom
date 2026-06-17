package axiom

type SuiteTestConfig struct {
	Runner *Runner
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
