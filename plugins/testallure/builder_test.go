package testallure_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/allure-framework/allure-go/commons/model"
	"github.com/allure-framework/allure-go/commons/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildAllureOptions_AllFields(t *testing.T) {
	cfg := &axiom.Config{
		Meta: axiom.Meta{
			Tags:        []string{"fast", "api"},
			Epic:        "Epic1",
			Suite:       "Suite1",
			Story:       "Story1",
			Layer:       "API",
			Platform:    "backend",
			Feature:     "Feature1",
			Severity:    axiom.SeverityCritical,
			SubSuite:    "SubSuite1",
			ParentSuite: "ParentSuite1",
			Issues:      []string{"AXIOM-42", "", "API/7"},
			TestCases:   []string{"", "USERS-001"},
			Labels: map[string]string{
				"team":  "backend",
				"owner": "nikita",
			},
		},
		Case: &axiom.Case{
			ID:          "ID123",
			Name:        "MyTest",
			Description: "Test description",
		},
	}

	result := recordResult(t, cfg)

	assert.Equal(t, "MyTest", result.Name)
	assert.Equal(t, "Test description", result.Description)
	assert.Equal(t, "ID123", result.TestCaseID)
	assertLabelValues(t, result.Labels, "ALLURE_ID", "ID123")
	assertLabelValues(t, result.Labels, "tag", "fast", "api")
	assertLabelValues(t, result.Labels, "epic", "Epic1")
	assertLabelValues(t, result.Labels, "suite", "Suite1")
	assertLabelValues(t, result.Labels, "story", "Story1")
	assertLabelValues(t, result.Labels, "layer", "API")
	assertLabelValues(t, result.Labels, "platform", "backend")
	assertLabelValues(t, result.Labels, "feature", "Feature1")
	assertLabelValues(t, result.Labels, "severity", "critical")
	assertLabelValues(t, result.Labels, "subSuite", "SubSuite1")
	assertLabelValues(t, result.Labels, "parentSuite", "ParentSuite1")
	assertLabelValues(t, result.Labels, "team", "backend")
	assertLabelValues(t, result.Labels, "owner", "nikita")
	assert.ElementsMatch(t, []model.Link{
		{
			Name: "AXIOM-42",
			URL:  "AXIOM-42",
			Type: string(model.LinkTypeIssue),
		},
		{
			Name: "API/7",
			URL:  "API/7",
			Type: string(model.LinkTypeIssue),
		},
		{
			Name: "USERS-001",
			URL:  "USERS-001",
			Type: string(model.LinkTypeTMS),
		},
	}, result.Links)
}

func TestBuildAllureOptions_EmptyConfig(t *testing.T) {
	assert.Empty(t, testallure.BuildAllureOptions(&axiom.Config{}))
}

func TestBuildAllureOptions_OnlyLabels(t *testing.T) {
	cfg := &axiom.Config{
		Meta: axiom.Meta{
			Labels: map[string]string{
				"a": "1",
				"b": "2",
			},
		},
	}

	result := recordResult(t, cfg)

	assertLabelValues(t, result.Labels, "a", "1")
	assertLabelValues(t, result.Labels, "b", "2")
}

func TestBuildAllureOptions_SeverityConversion(t *testing.T) {
	cfg := &axiom.Config{
		Meta: axiom.Meta{
			Severity: axiom.SeverityMinor,
		},
	}

	result := recordResult(t, cfg)

	assertLabelValues(t, result.Labels, "severity", "minor")
}

func recordResult(t *testing.T, cfg *axiom.Config) model.TestResult {
	t.Helper()

	memoryWriter := writer.NewInMemoryWriter()
	options := append(
		testallure.BuildAllureOptions(cfg),
		allure.WithWriter(memoryWriter),
	)

	allure.Wrap(t, func(*allure.Context) {}, options...)

	snapshot := memoryWriter.Snapshot()
	require.Len(t, snapshot.Results, 1)
	return snapshot.Results[0]
}

func assertLabelValues(t *testing.T, labels []model.Label, name string, expected ...string) {
	t.Helper()

	var actual []string
	for _, label := range labels {
		if label.Name == name {
			actual = append(actual, label.Value)
		}
	}

	assert.ElementsMatch(t, expected, actual, "label %q", name)
}
