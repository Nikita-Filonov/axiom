package axiom

type AllHook func(r *Runner)
type TestHook func(cfg *Config)
type StepHook func(cfg *Config, name string)

type Hooks struct {
	BeforeAll  []AllHook
	AfterAll   []AllHook
	BeforeTest []TestHook
	AfterTest  []TestHook
	BeforeStep []StepHook
	AfterStep  []StepHook
}

type HooksOption func(h *Hooks)

func NewHooks(options ...HooksOption) Hooks {
	h := Hooks{}
	for _, option := range options {
		option(&h)
	}
	return h
}

func WithBeforeAll(hook AllHook) HooksOption {
	return func(h *Hooks) {
		h.BeforeAll = append(h.BeforeAll, hook)
	}
}

func WithAfterAll(hook AllHook) HooksOption {
	return func(h *Hooks) {
		h.AfterAll = append(h.AfterAll, hook)
	}
}

func WithBeforeTest(hook TestHook) HooksOption {
	return func(h *Hooks) {
		h.BeforeTest = append(h.BeforeTest, hook)
	}
}

func WithAfterTest(hook TestHook) HooksOption {
	return func(h *Hooks) {
		h.AfterTest = append(h.AfterTest, hook)
	}
}

func WithBeforeStep(hook StepHook) HooksOption {
	return func(h *Hooks) {
		h.BeforeStep = append(h.BeforeStep, hook)
	}
}

func WithAfterStep(hook StepHook) HooksOption {
	return func(h *Hooks) {
		h.AfterStep = append(h.AfterStep, hook)
	}
}

func (h *Hooks) ApplyBeforeAll(r *Runner) {
	for _, hook := range h.BeforeAll {
		hook(r)
	}
}

func (h *Hooks) ApplyAfterAll(r *Runner) {
	for _, hook := range h.AfterAll {
		hook(r)
	}
}

func (h *Hooks) ApplyBeforeStep(cfg *Config, name string) {
	for _, hook := range h.BeforeStep {
		hook(cfg, name)
	}
}

func (h *Hooks) ApplyAfterStep(cfg *Config, name string) {
	for _, hook := range h.AfterStep {
		hook(cfg, name)
	}
}

func (h *Hooks) ApplyBeforeTest(cfg *Config) {
	for _, hook := range h.BeforeTest {
		hook(cfg)
	}
}

func (h *Hooks) ApplyAfterTest(cfg *Config) {
	for _, hook := range h.AfterTest {
		hook(cfg)
	}
}

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
