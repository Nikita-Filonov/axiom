package axiom

import (
	"reflect"
	"testing"
)

type Suite struct {
	RootT  *testing.T
	SubT   *testing.T
	Runner *Runner
}

type TestingSuite interface {
	SetRootT(*testing.T)
	SetSubT(*testing.T)
	SetRunner(*Runner)
	RunCase(Case, TestAction)
}

type BoundSuite[T TestingSuite] struct {
	rootT   *testing.T
	suite   T
	factory func() T
	config  SuiteConfig
	tests   []boundSuiteTest[T]
	ran     bool
}

type boundSuiteTest[T TestingSuite] struct {
	name   string
	action func(T)
	config SuiteTestConfig
}

func NewSuite[T TestingSuite](t *testing.T, suite T, options ...SuiteConfigOption) *BoundSuite[T] {
	if t == nil {
		panic("suite: nil *testing.T")
	}
	validateSuiteInstance(suite)

	cfg := NewSuiteConfig(options...)
	if cfg.Parallel {
		panic("suite: parallel suite tests require NewSuiteFactory")
	}

	suite.SetRootT(t)
	suite.SetSubT(nil)
	suite.SetRunner(cfg.Runner)

	return &BoundSuite[T]{
		rootT:  t,
		suite:  suite,
		config: cfg,
		tests:  make([]boundSuiteTest[T], 0),
	}
}

func NewSuiteFactory[T TestingSuite](t *testing.T, factory func() T, options ...SuiteConfigOption) *BoundSuite[T] {
	if t == nil {
		panic("suite: nil *testing.T")
	}
	if factory == nil {
		panic("suite: nil suite factory")
	}

	return &BoundSuite[T]{
		rootT:   t,
		factory: factory,
		config:  NewSuiteConfig(options...),
		tests:   make([]boundSuiteTest[T], 0),
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

func (s *BoundSuite[T]) Test(name string, action func(T), options ...SuiteTestConfigOption) {
	if s == nil {
		panic("suite: nil BoundSuite")
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

	cfg := NewSuiteTestConfig(options...)
	if cfg.Parallel && s.factory == nil {
		panic("suite: parallel suite tests require NewSuiteFactory")
	}

	s.tests = append(s.tests, boundSuiteTest[T]{
		name:   name,
		action: action,
		config: cfg,
	})
}

func (s *BoundSuite[T]) Run() {
	if s == nil {
		panic("suite: nil BoundSuite")
	}
	if s.ran {
		panic("suite: suite already ran")
	}
	s.ran = true

	s.config.Runner.ApplyStart()
	s.rootT.Cleanup(s.config.Runner.ApplyFinish)

	for _, test := range s.tests {
		s.rootT.Run(test.name, func(st *testing.T) {
			runner := test.config.Runner
			if runner == nil {
				runner = s.config.Runner
			}
			parallel := s.config.Parallel || test.config.Parallel

			runner.ApplyStart()
			s.rootT.Cleanup(runner.ApplyFinish)

			if parallel {
				st.Parallel()
			}

			suite := s.BuildSuite()
			suite.SetSubT(st)
			suite.SetRunner(runner)

			defer suite.SetSubT(nil)
			defer suite.SetRunner(s.config.Runner)

			test.action(suite)
		})
	}
}

func (s *BoundSuite[T]) BuildSuite() T {
	if s == nil {
		panic("suite: nil BoundSuite")
	}
	if s.factory == nil {
		return s.suite
	}

	suite := s.factory()
	validateSuiteInstance(suite)

	suite.SetRootT(s.rootT)
	suite.SetSubT(nil)
	suite.SetRunner(s.config.Runner)

	return suite
}

func (s *Suite) T() *testing.T {
	if s.SubT != nil {
		return s.SubT
	}

	return s.RootT
}

func (s *Suite) SetRootT(t *testing.T) {
	if s == nil {
		panic("suite: nil Suite")
	}

	s.RootT = t
}

func (s *Suite) SetSubT(t *testing.T) {
	if s == nil {
		panic("suite: nil Suite")
	}

	s.SubT = t
}

func (s *Suite) SetRunner(runner *Runner) {
	if s == nil {
		panic("suite: nil Suite")
	}

	s.Runner = runner
}

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
