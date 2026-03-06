package axiom

type TestAction func(cfg *Config)
type StepAction func()
type SetupAction func()
type TeardownAction func()

type WrapTestAction func(next TestAction) TestAction
type WrapStepAction func(name string, next StepAction) StepAction
type WrapSetupAction func(name string, next SetupAction) SetupAction
type WrapTeardownAction func(name string, next TeardownAction) TeardownAction

type SinkLogAction func(l Log)
type SinkAssertAction func(a Assert)
type SinkArtefactAction func(a Artefact)

type Runtime struct {
	TestWraps     []WrapTestAction
	StepWraps     []WrapStepAction
	SetupWraps    []WrapSetupAction
	TeardownWraps []WrapTeardownAction

	LogSinks      []SinkLogAction
	AssertSinks   []SinkAssertAction
	ArtefactSinks []SinkArtefactAction
}

type RuntimeOption func(*Runtime)

func NewRuntime(options ...RuntimeOption) Runtime {
	r := Runtime{}
	for _, option := range options {
		option(&r)
	}

	return r
}

func WithRuntimeTestWrap(w WrapTestAction) RuntimeOption {
	return func(r *Runtime) { r.EmitTestWrap(w) }
}

func WithRuntimeStepWrap(w WrapStepAction) RuntimeOption {
	return func(r *Runtime) { r.EmitStepWrap(w) }
}

func WithRuntimeSetupWrap(w WrapSetupAction) RuntimeOption {
	return func(r *Runtime) { r.EmitSetupWrap(w) }
}

func WithRuntimeTeardownWrap(w WrapTeardownAction) RuntimeOption {
	return func(r *Runtime) { r.EmitTeardownWrap(w) }
}

func WithRuntimeLogSink(s SinkLogAction) RuntimeOption {
	return func(r *Runtime) { r.EmitLogSink(s) }
}

func WithRuntimeAssertSink(s SinkAssertAction) RuntimeOption {
	return func(r *Runtime) { r.EmitAssertSink(s) }
}

func WithRuntimeArtefactSink(s SinkArtefactAction) RuntimeOption {
	return func(r *Runtime) { r.EmitArtefactSink(s) }
}

func (r *Runtime) EmitTestWrap(w WrapTestAction) {
	if w == nil {
		return
	}
	r.TestWraps = append(r.TestWraps, w)
}

func (r *Runtime) EmitStepWrap(w WrapStepAction) {
	if w == nil {
		return
	}
	r.StepWraps = append(r.StepWraps, w)
}

func (r *Runtime) EmitSetupWrap(w WrapSetupAction) {
	if w == nil {
		return
	}
	r.SetupWraps = append(r.SetupWraps, w)
}

func (r *Runtime) EmitTeardownWrap(w WrapTeardownAction) {
	if w == nil {
		return
	}
	r.TeardownWraps = append(r.TeardownWraps, w)
}

func (r *Runtime) EmitLogSink(s SinkLogAction) {
	if s == nil {
		return
	}
	r.LogSinks = append(r.LogSinks, s)
}

func (r *Runtime) EmitAssertSink(s SinkAssertAction) {
	if s == nil {
		return
	}
	r.AssertSinks = append(r.AssertSinks, s)
}

func (r *Runtime) EmitArtefactSink(s SinkArtefactAction) {
	if s == nil {
		return
	}
	r.ArtefactSinks = append(r.ArtefactSinks, s)
}

func (r *Runtime) Step(name string, fn func()) {
	wrapped := fn
	for i := len(r.StepWraps) - 1; i >= 0; i-- {
		wrapped = r.StepWraps[i](name, wrapped)
	}

	wrapped()
}

func (r *Runtime) Test(c *Config, action TestAction) {
	wrapped := action
	for i := len(r.TestWraps) - 1; i >= 0; i-- {
		wrapped = r.TestWraps[i](wrapped)
	}

	wrapped(c)
}

func (r *Runtime) Setup(name string, fn func()) {
	wrapped := fn
	for i := len(r.SetupWraps) - 1; i >= 0; i-- {
		wrapped = r.SetupWraps[i](name, wrapped)
	}
	wrapped()
}

func (r *Runtime) Teardown(name string, fn func()) {
	wrapped := fn
	for i := len(r.TeardownWraps) - 1; i >= 0; i-- {
		wrapped = r.TeardownWraps[i](name, wrapped)
	}
	wrapped()
}

func (r *Runtime) Log(l Log) {
	for _, sink := range r.LogSinks {
		sink(l)
	}
}

func (r *Runtime) Assert(a Assert) {
	for _, sink := range r.AssertSinks {
		sink(a)
	}
}

func (r *Runtime) Artefact(a Artefact) {
	for _, sink := range r.ArtefactSinks {
		sink(a)
	}
}

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
	if r.AssertSinks != nil {
		result.AssertSinks = append([]SinkAssertAction{}, r.AssertSinks...)
	}
	if r.ArtefactSinks != nil {
		result.ArtefactSinks = append([]SinkArtefactAction{}, r.ArtefactSinks...)
	}

	return result
}

func (r *Runtime) Join(other Runtime) Runtime {
	result := r.Copy()

	return Runtime{
		TestWraps:     append(result.TestWraps, other.TestWraps...),
		StepWraps:     append(result.StepWraps, other.StepWraps...),
		SetupWraps:    append(result.SetupWraps, other.SetupWraps...),
		TeardownWraps: append(result.TeardownWraps, other.TeardownWraps...),

		LogSinks:      append(result.LogSinks, other.LogSinks...),
		AssertSinks:   append(result.AssertSinks, other.AssertSinks...),
		ArtefactSinks: append(result.ArtefactSinks, other.ArtefactSinks...),
	}
}
