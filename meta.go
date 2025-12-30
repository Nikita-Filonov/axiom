package axiom

type Severity string

const (
	SeverityBlocker  Severity = "blocker"
	SeverityCritical Severity = "critical"
	SeverityNormal   Severity = "normal"
	SeverityMinor    Severity = "minor"
	SeverityTrivial  Severity = "trivial"
)

type Meta struct {
	Epic        string
	Tags        []string
	Suite       string
	Story       string
	Layer       string
	Issues      []string
	Labels      map[string]string
	Feature     string
	Severity    Severity
	SubSuite    string
	TestCases   []string
	ParentSuite string
}

type MetaOption func(*Meta)

func NewMeta(options ...MetaOption) Meta {
	m := Meta{}
	for _, option := range options {
		option(&m)
	}

	return m
}

func WithMetaEpic(epic string) MetaOption {
	return func(m *Meta) { m.Epic = epic }
}

func WithMetaSuite(suite string) MetaOption {
	return func(m *Meta) { m.Suite = suite }
}

func WithMetaStory(story string) MetaOption {
	return func(m *Meta) { m.Story = story }
}

func WithMetaLayer(layer string) MetaOption {
	return func(m *Meta) { m.Layer = layer }
}

func WithMetaFeature(feature string) MetaOption {
	return func(m *Meta) { m.Feature = feature }
}

func WithMetaSeverity(severity Severity) MetaOption {
	return func(m *Meta) { m.Severity = severity }
}

func WithMetaSubSuite(subSuite string) MetaOption {
	return func(m *Meta) { m.SubSuite = subSuite }
}

func WithMetaParentSuite(parentSuite string) MetaOption {
	return func(m *Meta) { m.ParentSuite = parentSuite }
}

func WithMetaTag(tag string) MetaOption {
	return func(m *Meta) { m.Tags = append(m.Tags, tag) }
}

func WithMetaTags(tags ...string) MetaOption {
	return func(m *Meta) { m.Tags = append(m.Tags, tags...) }
}

func WithMetaIssue(issue string) MetaOption {
	return func(m *Meta) { m.Issues = append(m.Issues, issue) }
}

func WithMetaIssues(issues ...string) MetaOption {
	return func(m *Meta) { m.Issues = append(m.Issues, issues...) }
}

func WithMetaLabel(key, value string) MetaOption {
	return func(m *Meta) {
		if m.Labels == nil {
			m.Labels = map[string]string{}
		}
		m.Labels[key] = value
	}
}

func WithMetaLabels(labels map[string]string) MetaOption {
	return func(m *Meta) {
		if m.Labels == nil {
			m.Labels = map[string]string{}
		}
		for k, v := range labels {
			m.Labels[k] = v
		}
	}
}

func WithMetaTestCase(testCase string) MetaOption {
	return func(m *Meta) { m.TestCases = append(m.TestCases, testCase) }
}

func WithMetaTestCases(testCases []string) MetaOption {
	return func(m *Meta) { m.TestCases = append(m.TestCases, testCases...) }
}

func (m *Meta) Join(other Meta) Meta {
	result := Meta{
		Epic:        m.Epic,
		Tags:        append([]string{}, m.Tags...),
		Suite:       m.Suite,
		Story:       m.Story,
		Layer:       m.Layer,
		Issues:      append([]string{}, m.Issues...),
		Labels:      map[string]string{},
		Feature:     m.Feature,
		Severity:    m.Severity,
		SubSuite:    m.SubSuite,
		TestCases:   append([]string{}, m.TestCases...),
		ParentSuite: m.ParentSuite,
	}
	for k, v := range m.Labels {
		result.Labels[k] = v
	}

	if other.Epic != "" {
		result.Epic = other.Epic
	}
	if other.Suite != "" {
		result.Suite = other.Suite
	}
	if other.Story != "" {
		result.Story = other.Story
	}
	if other.Layer != "" {
		result.Layer = other.Layer
	}
	if other.Feature != "" {
		result.Feature = other.Feature
	}
	if other.Severity != "" {
		result.Severity = other.Severity
	}
	if other.SubSuite != "" {
		result.SubSuite = other.SubSuite
	}
	if other.ParentSuite != "" {
		result.ParentSuite = other.ParentSuite
	}

	result.Tags = append(result.Tags, other.Tags...)
	result.Issues = append(result.Issues, other.Issues...)
	result.TestCases = append(result.TestCases, other.TestCases...)

	for k, v := range other.Labels {
		result.Labels[k] = v
	}

	return result
}

func (m *Meta) Normalize() {
	if m.Tags == nil {
		m.Tags = []string{}
	}
	if m.Issues == nil {
		m.Issues = []string{}
	}
	if m.Labels == nil {
		m.Labels = map[string]string{}
	}
	if m.Severity == "" {
		m.Severity = SeverityNormal
	}
	if m.TestCases == nil {
		m.TestCases = []string{}
	}
}
