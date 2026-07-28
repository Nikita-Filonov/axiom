package testallure_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
	"github.com/allure-framework/allure-go/commons/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_WritesAllureResultsToFilesystem(t *testing.T) {
	resultsDir := filepath.Join(t.TempDir(), "allure-results")
	t.Setenv("ALLURE_RESULTS_DIR", resultsDir)

	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(testallure.Plugin()),
	)
	testCase := axiom.NewCase(
		axiom.WithCaseID("AUTH-42"),
		axiom.WithCaseName("user can login"),
		axiom.WithCaseDescription("Valid credentials create a session."),
		axiom.WithCaseMeta(
			axiom.WithMetaEpic("authentication"),
			axiom.WithMetaFeature("login"),
			axiom.WithMetaSeverity(axiom.SeverityCritical),
			axiom.WithMetaTag("e2e"),
			axiom.WithMetaIssue("AXIOM-42"),
			axiom.WithMetaTestCase("USERS-001"),
		),
	)

	runner.RunCase(t, testCase, func(cfg *axiom.Config) {
		cfg.Setup("prepare user", func() {})
		cfg.Step("send login request", func() {
			cfg.Artefact(axiom.Artefact{
				Name: "request.json",
				Type: axiom.ArtefactTypeJSON,
				Data: []byte(`{"user":"alice"}`),
			})
			cfg.Step("validate response", func() {
				cfg.Artefact(axiom.Artefact{
					Name: "response.txt",
					Type: axiom.ArtefactTypeText,
					Data: []byte("status=200"),
				})
			})
		})
		cfg.Teardown("remove user", func() {})
	})

	result := readAllureResult(t, resultsDir)

	assert.NotEmpty(t, result.UUID)
	assert.NotEmpty(t, result.FullName)
	assert.Equal(t, "AUTH-42", result.TestCaseID)
	assert.NotEmpty(t, result.HistoryID)
	assert.Equal(t, "user can login", result.Name)
	assert.Equal(t, "Valid credentials create a session.", result.Description)
	assert.Equal(t, model.StatusPassed, result.Status)
	assert.Equal(t, model.StageFinished, result.Stage)
	assert.Positive(t, result.Start)
	assert.GreaterOrEqual(t, result.Stop, result.Start)
	assertLabelValues(t, result.Labels, "ALLURE_ID", "AUTH-42")
	assertLabelValues(t, result.Labels, "epic", "authentication")
	assertLabelValues(t, result.Labels, "feature", "login")
	assertLabelValues(t, result.Labels, "severity", "critical")
	assertLabelValues(t, result.Labels, "tag", "e2e")
	assert.ElementsMatch(t, []model.Link{
		{
			Name: "AXIOM-42",
			URL:  "AXIOM-42",
			Type: string(model.LinkTypeIssue),
		},
		{
			Name: "USERS-001",
			URL:  "USERS-001",
			Type: string(model.LinkTypeTMS),
		},
	}, result.Links)

	require.Len(t, result.Steps, 3)
	assert.Equal(t, "prepare user", result.Steps[0].Name)
	assert.Equal(t, "send login request", result.Steps[1].Name)
	assert.Equal(t, "remove user", result.Steps[2].Name)

	requestStep := result.Steps[1]
	require.Len(t, requestStep.Attachments, 1)
	requestAttachment := requestStep.Attachments[0]
	assert.Equal(t, "request.json", requestAttachment.Name)
	assert.Equal(t, "application/json", requestAttachment.Type)
	assert.JSONEq(t, `{"user":"alice"}`, string(readAllureAttachment(t, resultsDir, requestAttachment)))

	require.Len(t, requestStep.Steps, 1)
	responseStep := requestStep.Steps[0]
	assert.Equal(t, "validate response", responseStep.Name)
	require.Len(t, responseStep.Attachments, 1)
	responseAttachment := responseStep.Attachments[0]
	assert.Equal(t, "response.txt", responseAttachment.Name)
	assert.Equal(t, "text/plain", responseAttachment.Type)
	assert.Equal(t, "status=200", string(readAllureAttachment(t, resultsDir, responseAttachment)))

	entries, err := os.ReadDir(resultsDir)
	require.NoError(t, err)
	assert.Len(t, entries, 3, "one result and two attachment files expected")
}

func readAllureResult(t *testing.T, resultsDir string) model.TestResult {
	t.Helper()

	entries, err := os.ReadDir(resultsDir)
	require.NoError(t, err)

	var resultFiles []string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), "-result.json") {
			resultFiles = append(resultFiles, entry.Name())
		}
	}
	require.Len(t, resultFiles, 1)

	data, err := os.ReadFile(filepath.Join(resultsDir, resultFiles[0]))
	require.NoError(t, err)

	var result model.TestResult
	require.NoError(t, json.Unmarshal(data, &result))
	return result
}

func readAllureAttachment(
	t *testing.T,
	resultsDir string,
	attachment model.Attachment,
) []byte {
	t.Helper()

	require.NotEmpty(t, attachment.Source)
	data, err := os.ReadFile(filepath.Join(resultsDir, attachment.Source))
	require.NoError(t, err)
	assert.Equal(t, attachment.Size, int64(len(data)))
	return data
}
