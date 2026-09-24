package axiom

// AllHook runs once at a runner lifecycle boundary.
type AllHook func(r *Runner)

// TestHook runs before or after an individual test attempt.
type TestHook func(cfg *Config)

// StepHook runs before or after a named step.
type StepHook func(cfg *Config, name string)

// Hooks holds ordered lifecycle callbacks. Case test and step hooks run after
// the corresponding Runner hooks. BeforeAll and AfterAll are runner scoped;
// placing them on a Case has no effect during case execution.
type Hooks struct {
	BeforeAll  []AllHook
	AfterAll   []AllHook
	BeforeTest []TestHook
	AfterTest  []TestHook
	BeforeStep []StepHook
	AfterStep  []StepHook
}

// HooksOption registers a callback in Hooks.
type HooksOption func(h *Hooks)

// NewHooks returns Hooks configured by options.
func NewHooks(options ...HooksOption) Hooks {
	h := Hooks{}
	for _, option := range options {
		option(&h)
	}
	return h
}

// WithBeforeAll appends a hook run once when a Runner starts.
func WithBeforeAll(hook AllHook) HooksOption {
	return func(h *Hooks) {
		h.BeforeAll = append(h.BeforeAll, hook)
	}
}

// WithAfterAll appends a hook run once when a Runner finishes.
func WithAfterAll(hook AllHook) HooksOption {
	return func(h *Hooks) {
		h.AfterAll = append(h.AfterAll, hook)
	}
}

// WithBeforeTest appends a hook run before each test attempt body.
func WithBeforeTest(hook TestHook) HooksOption {
	return func(h *Hooks) {
		h.BeforeTest = append(h.BeforeTest, hook)
	}
}

// WithAfterTest appends a hook run after each attempt body and before fixture
// cleanup, including when the body panics.
func WithAfterTest(hook TestHook) HooksOption {
	return func(h *Hooks) {
		h.AfterTest = append(h.AfterTest, hook)
	}
}

// WithBeforeStep appends a hook run before each named step body.
func WithBeforeStep(hook StepHook) HooksOption {
	return func(h *Hooks) {
		h.BeforeStep = append(h.BeforeStep, hook)
	}
}

// WithAfterStep appends a hook run after each named step body, including when
// the body panics.
func WithAfterStep(hook StepHook) HooksOption {
	return func(h *Hooks) {
		h.AfterStep = append(h.AfterStep, hook)
	}
}

// ApplyBeforeAll calls runner start hooks in registration order.
func (h *Hooks) ApplyBeforeAll(r *Runner) {
	for _, hook := range h.BeforeAll {
		hook(r)
	}
}

// ApplyAfterAll calls runner finish hooks in registration order.
func (h *Hooks) ApplyAfterAll(r *Runner) {
	for _, hook := range h.AfterAll {
		hook(r)
	}
}

// ApplyBeforeStep calls step start hooks in registration order.
func (h *Hooks) ApplyBeforeStep(cfg *Config, name string) {
	for _, hook := range h.BeforeStep {
		hook(cfg, name)
	}
}

// ApplyAfterStep calls step finish hooks in registration order.
func (h *Hooks) ApplyAfterStep(cfg *Config, name string) {
	for _, hook := range h.AfterStep {
		hook(cfg, name)
	}
}

// ApplyBeforeTest calls test start hooks in registration order.
func (h *Hooks) ApplyBeforeTest(cfg *Config) {
	for _, hook := range h.BeforeTest {
		hook(cfg)
	}
}

// ApplyAfterTest calls test finish hooks in registration order.
func (h *Hooks) ApplyAfterTest(cfg *Config) {
	for _, hook := range h.AfterTest {
		hook(cfg)
	}
}

// Copy returns Hooks with independent callback slices.
func (h *Hooks) Copy() Hooks {
	var result Hooks

	if h.BeforeAll != nil {
		result.BeforeAll = append([]AllHook{}, h.BeforeAll...)
	}
	if h.AfterAll != nil {
		result.AfterAll = append([]AllHook{}, h.AfterAll...)
	}
	if h.BeforeTest != nil {
		result.BeforeTest = append([]TestHook{}, h.BeforeTest...)
	}
	if h.AfterTest != nil {
		result.AfterTest = append([]TestHook{}, h.AfterTest...)
	}
	if h.BeforeStep != nil {
		result.BeforeStep = append([]StepHook{}, h.BeforeStep...)
	}
	if h.AfterStep != nil {
		result.AfterStep = append([]StepHook{}, h.AfterStep...)
	}

	return result
}

// Join returns a copy with each callback list from other appended after h.
func (h *Hooks) Join(other Hooks) Hooks {
	result := h.Copy()

	return Hooks{
		BeforeAll:  append(result.BeforeAll, other.BeforeAll...),
		AfterAll:   append(result.AfterAll, other.AfterAll...),
		BeforeTest: append(result.BeforeTest, other.BeforeTest...),
		AfterTest:  append(result.AfterTest, other.AfterTest...),
		BeforeStep: append(result.BeforeStep, other.BeforeStep...),
		AfterStep:  append(result.AfterStep, other.AfterStep...),
	}
}
