package axiom

import "testing"

// This file exposes unexported internals to the axiom_test package.

// Runner lifecycle.

func (r *Runner) BuildConfig(t *testing.T, c *Case) *Config { return r.buildConfig(t, c, Execution{}) }

func (r *Runner) ApplyStart() { r.applyStart() }

func (r *Runner) ApplyFinish() { r.applyFinish() }

// Execution.

var NewExecution = newExecution

func (e Execution) NextAttempt() Execution { return e.nextAttempt() }

var NewCaseExecution = newCaseExecution

func (e *caseExecution) BaseConfig() *Config { return e.baseConfig }

func (e *caseExecution) CaseTemplate() Case { return e.caseTemplate }

func (e *caseExecution) NewConfig(execution Execution) *Config { return e.newConfig(execution) }

func (e *caseExecution) RunAttempts(t *testing.T, policies ...executionPolicy) {
	e.runAttempts(t, policies...)
}

func (e *caseExecution) WaitBeforeAttempt(attempt int) { e.waitBeforeAttempt(attempt) }

// Config and hooks.

func (c *Config) ApplyPlugins() { c.applyPlugins() }

func (c *Config) Test(action TestAction) { c.test(action) }

func (h *Hooks) ApplyBeforeAll(r *Runner) { h.applyBeforeAll(r) }

func (h *Hooks) ApplyAfterAll(r *Runner) { h.applyAfterAll(r) }

func (h *Hooks) ApplyBeforeTest(cfg *Config) { h.applyBeforeTest(cfg) }

func (h *Hooks) ApplyAfterTest(cfg *Config) { h.applyAfterTest(cfg) }

func (h *Hooks) ApplyBeforeStep(cfg *Config, name string) { h.applyBeforeStep(cfg, name) }

func (h *Hooks) ApplyAfterStep(cfg *Config, name string) { h.applyAfterStep(cfg, name) }

// Fixtures state.

func (f *Fixtures) Teardown(cfg *Config) { f.teardown(cfg) }

func (f *Fixtures) Cache() map[string]any { return f.cache }

func (f *Fixtures) SetCache(cache map[string]any) { f.cache = cache }

func (f *Fixtures) Cleanups() []func(*Config) { return f.cleanups }

func (f *Fixtures) SetCleanups(cleanups ...func(*Config)) { f.cleanups = cleanups }

// Resources state.

func (r *Resources) Teardown(runner *Runner) { r.teardown(runner) }

func (r *Resources) Cache() map[string]any { return r.cache }

func (r *Resources) SetCache(cache map[string]any) { r.cache = cache }

func (r *Resources) Cleanups() []func(*Runner) { return r.cleanups }

func (r *Resources) SetCleanups(cleanups ...func(*Runner)) { r.cleanups = cleanups }

// Runtime dispatch.

func (r *Runtime) Test(c *Config, action TestAction) { r.test(c, action) }

func (r *Runtime) Step(name string, fn func()) { r.step(name, fn) }

func (r *Runtime) Setup(name string, fn func()) { r.setup(name, fn) }

func (r *Runtime) Teardown(name string, fn func()) { r.teardown(name, fn) }

func (r *Runtime) Log(l Log) { r.log(l) }

func (r *Runtime) Event(e Event) { r.event(e) }

func (r *Runtime) Assert(a Assert) { r.assert(a) }

func (r *Runtime) Artefact(a Artefact) { r.artefact(a) }

// Events.

var (
	NewLogEvent      = newLogEvent
	NewAssertEvent   = newAssertEvent
	NewArtefactEvent = newArtefactEvent
)

// Suite.

var (
	NewSuiteConfig     = newSuiteConfig
	NewSuiteTestConfig = newSuiteTestConfig
)

func (s *SuiteRunner[T]) BuildSuite() T { return s.buildSuite() }

func (s *Suite) SetRootT(t *testing.T) { s.setRootT(t) }

func (s *Suite) SetSubT(t *testing.T) { s.setSubT(t) }

func (s *Suite) SetRunner(runner *Runner) { s.setRunner(runner) }

// ScalarTestingSuite is a TestingSuite that is not a struct. Only code inside
// the package can declare one, because TestingSuite has unexported methods.
type ScalarTestingSuite int

func (s *ScalarTestingSuite) setRootT(*testing.T) {}

func (s *ScalarTestingSuite) setSubT(*testing.T) {}

func (s *ScalarTestingSuite) setRunner(*Runner) {}

func (s *ScalarTestingSuite) RunCase(Case, TestAction) {}
