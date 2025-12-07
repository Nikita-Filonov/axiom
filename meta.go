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
	Epic     string
	Tags     []string
	Story    string
	Layer    string
	Labels   map[string]string
	Feature  string
	Severity Severity
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

func WithMetaTag(tag string) MetaOption {
	return func(m *Meta) {
		m.Tags = append(m.Tags, tag)
	}
}

func WithMetaTags(tags ...string) MetaOption {
	return func(m *Meta) {
		m.Tags = append(m.Tags, tags...)
	}
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

func (m *Meta) Join(other Meta) Meta {
	result := Meta{
		Epic:     m.Epic,
		Tags:     append([]string{}, m.Tags...),
		Story:    m.Story,
		Layer:    m.Layer,
		Labels:   map[string]string{},
		Feature:  m.Feature,
		Severity: m.Severity,
	}
	for k, v := range m.Labels {
		result.Labels[k] = v
	}

	if other.Epic != "" {
		result.Epic = other.Epic
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

	result.Tags = append(result.Tags, other.Tags...)

	for k, v := range other.Labels {
		result.Labels[k] = v
	}

	return result
}

func (m *Meta) Normalize() {
	if m.Tags == nil {
		m.Tags = []string{}
	}
	if m.Labels == nil {
		m.Labels = map[string]string{}
	}
	if m.Severity == "" {
		m.Severity = SeverityNormal
	}
}
