package testexplain

import "github.com/Nikita-Filonov/axiom"

type ExplanationKind string

const (
	ExplanationKindRunner ExplanationKind = "runner"
	ExplanationKindConfig ExplanationKind = "config"
)

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

type RunnerExplanation struct {
	Fixtures  []string            `json:"fixtures"`
	Resources []string            `json:"resources"`
	Plugins   CallableExplanation `json:"plugins"`
}

type CaseExplanation struct {
	ID          string              `json:"id,omitempty"`
	Name        string              `json:"name,omitempty"`
	Description string              `json:"description,omitempty"`
	ParamsType  string              `json:"paramsType,omitempty"`
	Fixtures    []string            `json:"fixtures"`
	Plugins     CallableExplanation `json:"plugins"`
}

type SkipExplanation struct {
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"`
}

type RetryExplanation struct {
	Times            int    `json:"times"`
	Delay            string `json:"delay"`
	DelayNanoseconds int64  `json:"delayNanoseconds"`
}

type ParallelExplanation struct {
	Enabled bool `json:"enabled"`
}

type ContextExplanation struct {
	Raw      bool     `json:"raw"`
	DB       bool     `json:"db"`
	MQ       bool     `json:"mq"`
	RPC      bool     `json:"rpc"`
	DataKeys []string `json:"dataKeys"`
}

type HooksExplanation struct {
	BeforeAll  CallableExplanation `json:"beforeAll"`
	AfterAll   CallableExplanation `json:"afterAll"`
	BeforeTest CallableExplanation `json:"beforeTest"`
	AfterTest  CallableExplanation `json:"afterTest"`
	BeforeStep CallableExplanation `json:"beforeStep"`
	AfterStep  CallableExplanation `json:"afterStep"`
}

type PluginsExplanation struct {
	Runner CallableExplanation `json:"runner"`
	Case   CallableExplanation `json:"case"`
	Total  int                 `json:"total"`
}

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

type CallableExplanation struct {
	Count int      `json:"count"`
	Names []string `json:"names,omitempty"`
}
