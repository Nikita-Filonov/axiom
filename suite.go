package axiom

import (
	"reflect"
	"testing"
)

// Suite is the embeddable implementation of [TestingSuite]. It gives registered
// suite tests access to their current *testing.T and selected Runner.
type Suite struct {
	RootT  *testing.T
	SubT   *testing.T
	Runner *Runner
}

// TestingSuite is the contract for values executed by [SuiteRunner]. A type
// satisfies it only by embedding Suite. Suite values must be non-nil pointers
// to structs.
type TestingSuite interface {
	setRootT(*testing.T)
	setSubT(*testing.T)
	setRunner(*Runner)
	RunCase(Case, TestAction)
}

// SuiteRunner registers related tests under one parent *testing.T and manages
// their runner lifecycle. Register tests with Test, then call Run once.
type SuiteRunner[T TestingSuite] struct {
	rootT   *testing.T
	suite   T
	factory func() T
	config  SuiteConfig
	tests   []suiteRunnerTest[T]
	ran     bool
}

type suiteRunnerTest[T TestingSuite] struct {
	name   string
	action func(T)
	config SuiteTestConfig
}

// NewSuite creates a SuiteRunner that reuses suite for its registered tests.
// Tests using this form run sequentially; use [NewSuiteFactory] for parallel
// suite tests. It panics if t or suite is nil or suite is not a pointer to a
// struct.
func NewSuite[T TestingSuite](t *testing.T, suite T, options ...SuiteConfigOption) *SuiteRunner[T] {
	if t == nil {
		panic("suite: nil *testing.T")
	}
	validateSuiteInstance(suite)

	cfg := newSuiteConfig(options...)
	if cfg.Parallel {
		panic("suite: parallel suite tests require NewSuiteFactory")
	}

	suite.setRootT(t)
	suite.setSubT(nil)
	suite.setRunner(cfg.Runner)

	return &SuiteRunner[T]{
		rootT:  t,
		suite:  suite,
		config: cfg,
		tests:  make([]suiteRunnerTest[T], 0),
	}
}

// NewSuiteFactory creates a SuiteRunner that calls factory for a fresh suite
// value for each registered test. This form supports parallel suite tests.
// It panics if t or factory is nil; each factory result must be a non-nil
// pointer to a struct implementing [TestingSuite].
func NewSuiteFactory[T TestingSuite](t *testing.T, factory func() T, options ...SuiteConfigOption) *SuiteRunner[T] {
	if t == nil {
		panic("suite: nil *testing.T")
	}
	if factory == nil {
		panic("suite: nil suite factory")
	}

	return &SuiteRunner[T]{
		rootT:   t,
		factory: factory,
		config:  newSuiteConfig(options...),
		tests:   make([]suiteRunnerTest[T], 0),
	}
}

func validateSuiteInstance(suite any) {
	suiteValue := reflect.ValueOf(suite)
	if !suiteValue.IsValid() {
		panic("suite: suite must be a non-nil pointer implementing axiom.TestingSuite")
	}
	if suiteValue.Kind() != reflect.Pointer {
		panic("suite: suite must be a non-nil pointer implementing axiom.TestingSuite")
	}
	if suiteValue.IsNil() {
		panic("suite: suite must be a non-nil pointer implementing axiom.TestingSuite")
	}
	if suiteValue.Elem().Kind() != reflect.Struct {
		panic("suite: suite must be a pointer to a struct implementing axiom.TestingSuite")
	}
}

// Test registers a named suite test. Names must be nonempty and unique within
// the SuiteRunner. Call it before Run.
func (s *SuiteRunner[T]) Test(name string, action func(T), options ...SuiteTestConfigOption) {
	if s == nil {
		panic("suite: nil SuiteRunner")
	}
	if s.ran {
		panic("suite: cannot register test after Run")
	}
	if name == "" {
		panic("suite: test name must not be empty")
	}
	if action == nil {
		panic("suite: nil test action")
	}
	for _, test := range s.tests {
		if test.name == name {
			panic("suite: duplicate test name: " + name)
		}
	}

	cfg := newSuiteTestConfig(options...)
	if cfg.Parallel && s.factory == nil {
		panic("suite: parallel suite tests require NewSuiteFactory")
	}

	s.tests = append(s.tests, suiteRunnerTest[T]{
		name:   name,
		action: action,
		config: cfg,
	})
}

// Run executes the registered suite tests and arranges runner cleanup after
// they finish. It may be called only once.
func (s *SuiteRunner[T]) Run() {
	if s == nil {
		panic("suite: nil SuiteRunner")
	}
	if s.ran {
		panic("suite: suite already ran")
	}
	s.ran = true

	s.config.Runner.applyStart()
	s.rootT.Cleanup(s.config.Runner.applyFinish)

	for _, test := range s.tests {
		s.rootT.Run(test.name, func(st *testing.T) {
			runner := test.config.Runner
			if runner == nil {
				runner = s.config.Runner
			}
			parallel := s.config.Parallel || test.config.Parallel

			runner.applyStart()
			s.rootT.Cleanup(runner.applyFinish)

			if parallel {
				st.Parallel()
			}

			suite := s.buildSuite()
			suite.setSubT(st)
			suite.setRunner(runner)

			defer suite.setSubT(nil)
			defer suite.setRunner(s.config.Runner)

			test.action(suite)
		})
	}
}

// buildSuite returns the configured suite instance or creates one with the
// factory. It panics if the SuiteRunner is nil.
func (s *SuiteRunner[T]) buildSuite() T {
	if s == nil {
		panic("suite: nil SuiteRunner")
	}
	if s.factory == nil {
		return s.suite
	}

	suite := s.factory()
	validateSuiteInstance(suite)

	suite.setRootT(s.rootT)
	suite.setSubT(nil)
	suite.setRunner(s.config.Runner)

	return suite
}

// T returns the active subtest, or the root test when no subtest is active.
func (s *Suite) T() *testing.T {
	if s.SubT != nil {
		return s.SubT
	}

	return s.RootT
}

// setRootT sets the test that owns the suite lifecycle.
func (s *Suite) setRootT(t *testing.T) {
	if s == nil {
		panic("suite: nil Suite")
	}

	s.RootT = t
}

// setSubT sets the currently active suite subtest.
func (s *Suite) setSubT(t *testing.T) {
	if s == nil {
		panic("suite: nil Suite")
	}

	s.SubT = t
}

// setRunner sets the runner used by subsequent suite cases.
func (s *Suite) setRunner(runner *Runner) {
	if s == nil {
		panic("suite: nil Suite")
	}

	s.Runner = runner
}

// RunCase executes c under the active suite subtest and runner.
func (s *Suite) RunCase(c Case, a TestAction) {
	if s == nil {
		panic("suite: nil Suite")
	}
	if s.Runner == nil {
		panic("suite: runner is not configured")
	}
	if s.SubT == nil {
		panic("suite: nil *testing.T")
	}

	s.Runner.runCase(s.SubT, c, a)
}
