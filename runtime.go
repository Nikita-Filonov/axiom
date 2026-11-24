package axiom

// TestAction is the body executed for one test attempt.
type TestAction func(cfg *Config)

// StepAction is the body of one named step.
type StepAction func()

// SetupAction is the body of one named setup operation.
type SetupAction func()

// TeardownAction is the body of one named teardown operation.
type TeardownAction func()

// WrapTestAction surrounds a selected attempt, including a declarative skip.
// Its continuation enters hooks and the body for DecisionRun, or calls native
// Skip for DecisionSkip. Excluded attempts never enter wrappers. Forward next
// at most once; use defer or native cleanup for work that must survive Goexit.
type WrapTestAction func(next TestAction) TestAction

// WrapStepAction wraps a named step action before it executes.
// The operation carries immutable values supplied by this invocation only.
type WrapStepAction func(operation Operation, next StepAction) StepAction

// WrapSetupAction wraps a named setup action before it executes.
// The operation carries immutable values supplied by this invocation only.
type WrapSetupAction func(operation Operation, next SetupAction) SetupAction

// WrapTeardownAction wraps a named teardown action before it executes.
// The operation carries immutable values supplied by this invocation only.
type WrapTeardownAction func(operation Operation, next TeardownAction) TeardownAction

// SinkLogAction receives a structured Log.
type SinkLogAction func(l Log)

// SinkEventAction receives a raw Event.
type SinkEventAction func(e Event)

// SinkAssertAction receives a structured Assert.
type SinkAssertAction func(a Assert)

// SinkAttemptAction receives an attempt after native completion or selection.
// It must be concurrency-safe and observational: do not run test actions, call
// FailNow/SkipNow, or register cleanup. The attempt's T has already finished and
// parallel results may arrive after RunCase returns. Capture metadata during
// plugin installation. Fatal process termination may prevent result delivery.
type SinkAttemptAction func(a Attempt)

// SinkArtefactAction receives an Artefact.
type SinkArtefactAction func(a Artefact)

// Runtime holds wrappers and sinks for test execution and emitted events.
// Runner and Case runtime settings are combined into each attempt's Config.
// Earlier wrappers are outermost; sinks receive values in registration order.
type Runtime struct {
	TestWraps     []WrapTestAction
	StepWraps     []WrapStepAction
	SetupWraps    []WrapSetupAction
	TeardownWraps []WrapTeardownAction

	LogSinks      []SinkLogAction
	EventSinks    []SinkEventAction
	AssertSinks   []SinkAssertAction
	AttemptSinks  []SinkAttemptAction
	ArtefactSinks []SinkArtefactAction
}

// RuntimeOption registers a wrapper or sink on a Runtime. Nil wrappers and
// sinks are ignored.
type RuntimeOption func(*Runtime)

// NewRuntime returns a Runtime with the supplied wrappers and sinks.
func NewRuntime(options ...RuntimeOption) Runtime {
	r := Runtime{}
	for _, option := range options {
		option(&r)
	}

	return r
}

// WithRuntimeTestWrap appends a wrapper around selected attempts, including skips.
func WithRuntimeTestWrap(w WrapTestAction) RuntimeOption {
	return func(r *Runtime) { r.EmitTestWrap(w) }
}

// WithRuntimeStepWrap appends a wrapper around named step actions.
func WithRuntimeStepWrap(w WrapStepAction) RuntimeOption {
	return func(r *Runtime) { r.EmitStepWrap(w) }
}

// WithRuntimeSetupWrap appends a wrapper around named setup actions.
func WithRuntimeSetupWrap(w WrapSetupAction) RuntimeOption {
	return func(r *Runtime) { r.EmitSetupWrap(w) }
}

// WithRuntimeTeardownWrap appends a wrapper around named teardown actions.
func WithRuntimeTeardownWrap(w WrapTeardownAction) RuntimeOption {
	return func(r *Runtime) { r.EmitTeardownWrap(w) }
}

// WithRuntimeLogSink appends a receiver for structured logs.
func WithRuntimeLogSink(s SinkLogAction) RuntimeOption {
	return func(r *Runtime) { r.EmitLogSink(s) }
}

// WithRuntimeEventSink appends a receiver for raw events.
func WithRuntimeEventSink(s SinkEventAction) RuntimeOption {
	return func(r *Runtime) { r.EmitEventSink(s) }
}

// WithRuntimeAssertSink appends a receiver for assertion facts.
func WithRuntimeAssertSink(s SinkAssertAction) RuntimeOption {
	return func(r *Runtime) { r.EmitAssertSink(s) }
}

// WithRuntimeAttemptSink appends a receiver of terminal attempts, including
// exclusions and native deselections. It does not receive execution phases.
func WithRuntimeAttemptSink(s SinkAttemptAction) RuntimeOption {
	return func(r *Runtime) { r.EmitAttemptSink(s) }
}

// WithRuntimeArtefactSink appends a receiver for artefacts.
func WithRuntimeArtefactSink(s SinkArtefactAction) RuntimeOption {
	return func(r *Runtime) { r.EmitArtefactSink(s) }
}

// EmitTestWrap appends a non-nil test wrapper.
func (r *Runtime) EmitTestWrap(w WrapTestAction) {
	if w == nil {
		return
	}
	r.TestWraps = append(r.TestWraps, w)
}

// EmitStepWrap appends a non-nil step wrapper.
func (r *Runtime) EmitStepWrap(w WrapStepAction) {
	if w == nil {
		return
	}
	r.StepWraps = append(r.StepWraps, w)
}

// EmitSetupWrap appends a non-nil setup wrapper.
func (r *Runtime) EmitSetupWrap(w WrapSetupAction) {
	if w == nil {
		return
	}
	r.SetupWraps = append(r.SetupWraps, w)
}

// EmitTeardownWrap appends a non-nil teardown wrapper.
func (r *Runtime) EmitTeardownWrap(w WrapTeardownAction) {
	if w == nil {
		return
	}
	r.TeardownWraps = append(r.TeardownWraps, w)
}

// EmitLogSink appends a non-nil log sink.
func (r *Runtime) EmitLogSink(s SinkLogAction) {
	if s == nil {
		return
	}
	r.LogSinks = append(r.LogSinks, s)
}

// EmitEventSink appends a non-nil event sink.
func (r *Runtime) EmitEventSink(s SinkEventAction) {
	if s == nil {
		return
	}
	r.EventSinks = append(r.EventSinks, s)
}

// EmitAssertSink appends a non-nil assertion sink.
func (r *Runtime) EmitAssertSink(s SinkAssertAction) {
	if s == nil {
		return
	}
	r.AssertSinks = append(r.AssertSinks, s)
}

// EmitAttemptSink appends a non-nil attempt receiver. Sinks receive independent
// snapshots in registration order and must obey SinkAttemptAction's contract.
func (r *Runtime) EmitAttemptSink(s SinkAttemptAction) {
	if s == nil {
		return
	}
	r.AttemptSinks = append(r.AttemptSinks, s)
}

// EmitArtefactSink appends a non-nil artefact sink.
func (r *Runtime) EmitArtefactSink(s SinkArtefactAction) {
	if s == nil {
		return
	}
	r.ArtefactSinks = append(r.ArtefactSinks, s)
}

// step runs a named action through registered step wrappers.
func (r *Runtime) step(operation Operation, fn func()) {
	wrapped := fn
	for i := len(r.StepWraps) - 1; i >= 0; i-- {
		wrapped = r.StepWraps[i](operation, wrapped)
	}

	wrapped()
}

// test runs an attempt through registered test wrappers.
func (r *Runtime) test(c *Config, action TestAction) {
	wrapped := action
	for i := len(r.TestWraps) - 1; i >= 0; i-- {
		wrapped = r.TestWraps[i](wrapped)
	}

	wrapped(c)
}

// setup runs a named action through registered setup wrappers.
func (r *Runtime) setup(operation Operation, fn func()) {
	wrapped := fn
	for i := len(r.SetupWraps) - 1; i >= 0; i-- {
		wrapped = r.SetupWraps[i](operation, wrapped)
	}
	wrapped()
}

// teardown runs a named action through registered teardown wrappers.
func (r *Runtime) teardown(operation Operation, fn func()) {
	wrapped := fn
	for i := len(r.TeardownWraps) - 1; i >= 0; i-- {
		wrapped = r.TeardownWraps[i](operation, wrapped)
	}
	wrapped()
}

// log sends a structured log to registered sinks.
func (r *Runtime) log(l Log) {
	for _, sink := range r.LogSinks {
		sink(l)
	}
}

// event sends a raw event to registered sinks.
func (r *Runtime) event(e Event) {
	for _, sink := range r.EventSinks {
		sink(e)
	}
}

// assert sends an assertion fact to registered sinks.
func (r *Runtime) assert(a Assert) {
	for _, sink := range r.AssertSinks {
		sink(a)
	}
}

// attempt sends independent terminal snapshots to registered sinks.
func (r *Runtime) attempt(a Attempt) {
	for _, sink := range r.AttemptSinks {
		sink(a.clone())
	}
}

// artefact sends an artefact to registered sinks.
func (r *Runtime) artefact(a Artefact) {
	for _, sink := range r.ArtefactSinks {
		sink(a)
	}
}

// Copy returns a Runtime with independent wrapper and sink slices.
func (r *Runtime) Copy() Runtime {
	var result Runtime

	if r.TestWraps != nil {
		result.TestWraps = append([]WrapTestAction{}, r.TestWraps...)
	}
	if r.StepWraps != nil {
		result.StepWraps = append([]WrapStepAction{}, r.StepWraps...)
	}
	if r.SetupWraps != nil {
		result.SetupWraps = append([]WrapSetupAction{}, r.SetupWraps...)
	}
	if r.TeardownWraps != nil {
		result.TeardownWraps = append([]WrapTeardownAction{}, r.TeardownWraps...)
	}

	if r.LogSinks != nil {
		result.LogSinks = append([]SinkLogAction{}, r.LogSinks...)
	}
	if r.EventSinks != nil {
		result.EventSinks = append([]SinkEventAction{}, r.EventSinks...)
	}
	if r.AssertSinks != nil {
		result.AssertSinks = append([]SinkAssertAction{}, r.AssertSinks...)
	}
	if r.AttemptSinks != nil {
		result.AttemptSinks = append([]SinkAttemptAction{}, r.AttemptSinks...)
	}
	if r.ArtefactSinks != nil {
		result.ArtefactSinks = append([]SinkArtefactAction{}, r.ArtefactSinks...)
	}

	return result
}

// Join returns a Runtime with other's wrappers and sinks appended after r's.
func (r *Runtime) Join(other Runtime) Runtime {
	result := r.Copy()

	return Runtime{
		TestWraps:     append(result.TestWraps, other.TestWraps...),
		StepWraps:     append(result.StepWraps, other.StepWraps...),
		SetupWraps:    append(result.SetupWraps, other.SetupWraps...),
		TeardownWraps: append(result.TeardownWraps, other.TeardownWraps...),

		LogSinks:      append(result.LogSinks, other.LogSinks...),
		EventSinks:    append(result.EventSinks, other.EventSinks...),
		AssertSinks:   append(result.AssertSinks, other.AssertSinks...),
		AttemptSinks:  append(result.AttemptSinks, other.AttemptSinks...),
		ArtefactSinks: append(result.ArtefactSinks, other.ArtefactSinks...),
	}
}
