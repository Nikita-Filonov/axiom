package axiom

type TestHook func(cfg *Config)
type StepHook func(cfg *Config, name string)

type Hooks struct {
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

func (h *Hooks) Join(other Hooks) Hooks {
	return Hooks{
		BeforeTest: append(h.BeforeTest, other.BeforeTest...),
		AfterTest:  append(h.AfterTest, other.AfterTest...),
		BeforeStep: append(h.BeforeStep, other.BeforeStep...),
		AfterStep:  append(h.AfterStep, other.AfterStep...),
	}
}
