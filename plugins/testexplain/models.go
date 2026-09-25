package testexplain

import "github.com/Nikita-Filonov/axiom"

// ExplanationKind identifies the source of a configuration snapshot.
type ExplanationKind string

// ExplanationKind values distinguish runner and attempt snapshots.
const (
	ExplanationKindRunner ExplanationKind = "runner"
	ExplanationKindConfig ExplanationKind = "config"
)

// Explanation is a readable snapshot of runner or attempt configuration.
type Explanation struct {
	Kind      ExplanationKind     `json:"kind"`
	Runner    *RunnerExplanation  `json:"runner,omitempty"`
	Case      *CaseExplanation    `json:"case,omitempty"`
	Meta      axiom.Meta          `json:"meta"`
	Skip      SkipExplanation     `json:"skip"`
	Retry     RetryExplanation    `json:"retry"`
	Parallel  ParallelExplanation `json:"parallel"`
	Context   ContextExplanation  `json:"context"`
	Fixtures  []string            `json:"fixtures"`
	Resources []string            `json:"resources"`
	Hooks     HooksExplanation    `json:"hooks"`
	Plugins   PluginsExplanation  `json:"plugins"`
	Runtime   RuntimeExplanation  `json:"runtime"`
}

// RunnerExplanation lists runner-level definitions and plugins.
type RunnerExplanation struct {
	Fixtures  []string            `json:"fixtures"`
	Resources []string            `json:"resources"`
	Plugins   CallableExplanation `json:"plugins"`
	Meta      axiom.Meta          `json:"meta"`
	Skip      SkipExplanation     `json:"skip"`
	Retry     RetryExplanation    `json:"retry"`
	Parallel  ParallelExplanation `json:"parallel"`
	Context   ContextExplanation  `json:"context"`
	Hooks     HooksExplanation    `json:"hooks"`
	Runtime   RuntimeExplanation  `json:"runtime"`
	Parent    *Explanation        `json:"parent,omitempty"`
	Overlay   *Explanation        `json:"overlay,omitempty"`
	Cycle     bool                `json:"cycle,omitempty"`
}

// CaseExplanation summarizes the selected case.
type CaseExplanation struct {
	ID          string              `json:"id,omitempty"`
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	ParamsType  string              `json:"paramsType,omitempty"`
	Fixtures    []string            `json:"fixtures"`
	Plugins     CallableExplanation `json:"plugins"`
	Meta        axiom.Meta          `json:"meta"`
	Skip        SkipExplanation     `json:"skip"`
	Retry       RetryExplanation    `json:"retry"`
	Parallel    ParallelExplanation `json:"parallel"`
	Context     ContextExplanation  `json:"context"`
	Hooks       HooksExplanation    `json:"hooks"`
	Runtime     RuntimeExplanation  `json:"runtime"`
}

// SkipExplanation summarizes the effective skip policy.
type SkipExplanation struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"`
}

// RetryExplanation summarizes the effective retry policy.
type RetryExplanation struct {
	Times            int    `json:"times"`
	Delay            string `json:"delay"`
	DelayNanoseconds int64  `json:"delayNanoseconds"`
}

// ParallelExplanation reports whether parallel execution is enabled.
type ParallelExplanation struct {
	Enabled bool `json:"enabled"`
}

// ContextExplanation reports available contexts and named data keys.
type ContextExplanation struct {
	Raw      bool     `json:"raw"`
	DB       bool     `json:"db"`
	MQ       bool     `json:"mq"`
	RPC      bool     `json:"rpc"`
	DataKeys []string `json:"dataKeys"`
}

// HooksExplanation counts and names registered lifecycle hooks.
type HooksExplanation struct {
	BeforeAll  CallableExplanation `json:"beforeAll"`
	AfterAll   CallableExplanation `json:"afterAll"`
	BeforeTest CallableExplanation `json:"beforeTest"`
	AfterTest  CallableExplanation `json:"afterTest"`
	BeforeStep CallableExplanation `json:"beforeStep"`
	AfterStep  CallableExplanation `json:"afterStep"`
}

// PluginsExplanation counts and names runner and case plugins.
type PluginsExplanation struct {
	Runner CallableExplanation `json:"runner"`
	Case   CallableExplanation `json:"case"`
	Total  int                 `json:"total"`
}

// RuntimeExplanation counts and names wrappers and sinks.
type RuntimeExplanation struct {
	TestWraps     CallableExplanation `json:"testWraps"`
	StepWraps     CallableExplanation `json:"stepWraps"`
	SetupWraps    CallableExplanation `json:"setupWraps"`
	TeardownWraps CallableExplanation `json:"teardownWraps"`
	LogSinks      CallableExplanation `json:"logSinks"`
	AssertSinks   CallableExplanation `json:"assertSinks"`
	ArtefactSinks CallableExplanation `json:"artefactSinks"`
	EventSinks    CallableExplanation `json:"eventSinks"`
}

// CallableExplanation counts callbacks and lists their names in registration order.
type CallableExplanation struct {
	Count int      `json:"count"`
	Names []string `json:"names,omitempty"`
}
