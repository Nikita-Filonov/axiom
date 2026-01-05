package axiom_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestNewMeta_Defaults(t *testing.T) {
	m := axiom.NewMeta()

	assert.Nil(t, m.Tags)
	assert.Nil(t, m.Labels)
	assert.Nil(t, m.Issues)
	assert.Nil(t, m.TestCases)

	assert.Empty(t, m.Epic)
	assert.Empty(t, m.Suite)
	assert.Empty(t, m.SubSuite)
	assert.Empty(t, m.ParentSuite)
	assert.Empty(t, m.Story)
	assert.Empty(t, m.Layer)
	assert.Empty(t, m.Feature)
	assert.Empty(t, m.Platform)

	assert.Equal(t, axiom.Severity(""), m.Severity)
}

func TestNewMeta_WithOptions(t *testing.T) {
	m := axiom.NewMeta(
		axiom.WithMetaEpic("Payments"),
		axiom.WithMetaSuite("API"),
		axiom.WithMetaSubSuite("Transfers"),
		axiom.WithMetaParentSuite("PaymentsRoot"),
		axiom.WithMetaStory("User can send money"),
		axiom.WithMetaLayer("Integration"),
		axiom.WithMetaFeature("Transfer"),
		axiom.WithMetaPlatform("Backend"),
		axiom.WithMetaSeverity(axiom.SeverityCritical),
		axiom.WithMetaTag("smoke"),
		axiom.WithMetaTags("fast", "api"),
		axiom.WithMetaIssue("ISSUE-1"),
		axiom.WithMetaIssues("ISSUE-2", "ISSUE-3"),
		axiom.WithMetaTestCase("TC-1"),
		axiom.WithMetaTestCases([]string{"TC-2", "TC-3"}),
		axiom.WithMetaLabel("team", "core"),
		axiom.WithMetaLabels(map[string]string{"env": "prod"}),
	)

	assert.Equal(t, "Payments", m.Epic)
	assert.Equal(t, "API", m.Suite)
	assert.Equal(t, "Transfers", m.SubSuite)
	assert.Equal(t, "PaymentsRoot", m.ParentSuite)
	assert.Equal(t, "User can send money", m.Story)
	assert.Equal(t, "Integration", m.Layer)
	assert.Equal(t, "Transfer", m.Feature)
	assert.Equal(t, "Backend", m.Platform)
	assert.Equal(t, axiom.SeverityCritical, m.Severity)

	assert.ElementsMatch(t, []string{"smoke", "fast", "api"}, m.Tags)
	assert.ElementsMatch(t, []string{"ISSUE-1", "ISSUE-2", "ISSUE-3"}, m.Issues)
	assert.ElementsMatch(t, []string{"TC-1", "TC-2", "TC-3"}, m.TestCases)

	assert.Equal(t, "core", m.Labels["team"])
	assert.Equal(t, "prod", m.Labels["env"])
}

func TestMetaNormalize_InitializesLabels(t *testing.T) {
	var m axiom.Meta
	assert.Nil(t, m.Labels)

	m.Normalize()

	assert.NotNil(t, m.Labels)
	assert.Len(t, m.Labels, 0)
}

func TestMetaNormalize_InitializesAllCollectionsAndSeverity(t *testing.T) {
	var m axiom.Meta

	m.Normalize()

	assert.NotNil(t, m.Tags)
	assert.NotNil(t, m.Issues)
	assert.NotNil(t, m.TestCases)
	assert.NotNil(t, m.Labels)

	assert.Len(t, m.Tags, 0)
	assert.Len(t, m.Issues, 0)
	assert.Len(t, m.TestCases, 0)
	assert.Len(t, m.Labels, 0)

	assert.Equal(t, axiom.SeverityNormal, m.Severity)
}

func TestMetaNormalize_DoesNotOverrideExistingLabels(t *testing.T) {
	m := axiom.Meta{
		Labels: map[string]string{"x": "1"},
	}

	m.Normalize()

	assert.Equal(t, "1", m.Labels["x"])
}

func TestMetaJoin_OverridesSimpleFields(t *testing.T) {
	base := axiom.Meta{
		Epic:     "BaseEpic",
		Story:    "BaseStory",
		Layer:    "BaseLayer",
		Feature:  "BaseFeature",
		Platform: "BasePlatform",
		Severity: axiom.SeverityNormal,
		Labels:   map[string]string{"a": "1"},
	}

	other := axiom.Meta{
		Epic:     "NewEpic",
		Story:    "NewStory",
		Layer:    "NewLayer",
		Feature:  "NewFeature",
		Platform: "NewPlatform",
		Severity: axiom.SeverityCritical,
		Labels:   map[string]string{"b": "2"},
	}

	result := base.Join(other)

	assert.Equal(t, "NewEpic", result.Epic)
	assert.Equal(t, "NewStory", result.Story)
	assert.Equal(t, "NewLayer", result.Layer)
	assert.Equal(t, "NewFeature", result.Feature)
	assert.Equal(t, "NewPlatform", result.Platform)
	assert.Equal(t, axiom.SeverityCritical, result.Severity)

	assert.Equal(t, "1", result.Labels["a"])
	assert.Equal(t, "2", result.Labels["b"])
}

func TestMetaJoin_OverridesSuites(t *testing.T) {
	base := axiom.Meta{
		Suite:       "BaseSuite",
		SubSuite:    "BaseSub",
		ParentSuite: "BaseParent",
		Labels:      map[string]string{},
	}

	other := axiom.Meta{
		Suite:       "NewSuite",
		SubSuite:    "NewSub",
		ParentSuite: "NewParent",
		Labels:      map[string]string{},
	}

	result := base.Join(other)

	assert.Equal(t, "NewSuite", result.Suite)
	assert.Equal(t, "NewSub", result.SubSuite)
	assert.Equal(t, "NewParent", result.ParentSuite)
}

func TestMetaJoin_TagsAreAppended(t *testing.T) {
	base := axiom.Meta{
		Tags:   []string{"a", "b"},
		Labels: map[string]string{},
	}

	other := axiom.Meta{
		Tags:   []string{"c", "d"},
		Labels: map[string]string{},
	}

	result := base.Join(other)

	assert.ElementsMatch(t, []string{"a", "b", "c", "d"}, result.Tags)
}

func TestMetaJoin_IssuesAndTestCasesAreAppended(t *testing.T) {
	base := axiom.Meta{
		Issues:    []string{"ISSUE-1"},
		TestCases: []string{"TC-1"},
		Labels:    map[string]string{},
	}

	other := axiom.Meta{
		Issues:    []string{"ISSUE-2"},
		TestCases: []string{"TC-2", "TC-3"},
		Labels:    map[string]string{},
	}

	result := base.Join(other)

	assert.ElementsMatch(t, []string{"ISSUE-1", "ISSUE-2"}, result.Issues)
	assert.ElementsMatch(t, []string{"TC-1", "TC-2", "TC-3"}, result.TestCases)
}

func TestMetaJoin_LabelOverrideNewKeys(t *testing.T) {
	base := axiom.Meta{
		Labels: map[string]string{"foo": "1"},
	}

	other := axiom.Meta{
		Labels: map[string]string{"bar": "2"},
	}

	result := base.Join(other)

	assert.Equal(t, "1", result.Labels["foo"])
	assert.Equal(t, "2", result.Labels["bar"])
}

func TestMetaJoin_LabelOverridesExistingKeys(t *testing.T) {
	base := axiom.Meta{
		Labels: map[string]string{"env": "dev"},
	}

	other := axiom.Meta{
		Labels: map[string]string{"env": "prod"},
	}

	result := base.Join(other)

	assert.Equal(t, "prod", result.Labels["env"])
}

func TestMetaJoin_DoesNotOverrideEmptyFields(t *testing.T) {
	base := axiom.Meta{
		Epic:     "A",
		Story:    "B",
		Layer:    "C",
		Feature:  "D",
		Platform: "E",
		Severity: axiom.SeverityMinor,
		Labels:   map[string]string{},
	}

	other := axiom.Meta{
		// all empty
		Labels: map[string]string{},
	}

	result := base.Join(other)

	assert.Equal(t, "A", result.Epic)
	assert.Equal(t, "B", result.Story)
	assert.Equal(t, "C", result.Layer)
	assert.Equal(t, "D", result.Feature)
	assert.Equal(t, "E", result.Platform)
	assert.Equal(t, axiom.SeverityMinor, result.Severity)
}

func TestMetaJoin_DoesNotOverrideEmptySuites(t *testing.T) {
	base := axiom.Meta{
		Suite:       "A",
		SubSuite:    "B",
		ParentSuite: "C",
		Labels:      map[string]string{},
	}

	other := axiom.Meta{
		Labels: map[string]string{},
	}

	result := base.Join(other)

	assert.Equal(t, "A", result.Suite)
	assert.Equal(t, "B", result.SubSuite)
	assert.Equal(t, "C", result.ParentSuite)
}
