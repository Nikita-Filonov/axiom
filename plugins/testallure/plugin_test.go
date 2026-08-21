package testallure_test

import (
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/allure-framework/allure-go/commons/model"
	"github.com/allure-framework/allure-go/commons/writer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_AddsAllExpectedRuntimeHooks(t *testing.T) {
	cfg := &axiom.Config{SubT: t}

	testallure.Plugin()(cfg)

	assert.Len(t, cfg.Runtime.TestWraps, 1)
	assert.Len(t, cfg.Runtime.StepWraps, 1)
	assert.Len(t, cfg.Runtime.SetupWraps, 1)
	assert.Len(t, cfg.Runtime.TeardownWraps, 1)
	assert.Len(t, cfg.Runtime.ArtefactSinks, 1)
}

func TestPlugin_StepOutsideAllureTestStillRuns(t *testing.T) {
	cfg := &axiom.Config{SubT: t}
	testallure.Plugin()(cfg)

	called := false
	cfg.Runtime.Step("outside test", func() {
		called = true
	})

	assert.True(t, called)
}

func TestPlugin_ReportsTestStepsAndArtefacts(t *testing.T) {
	memoryWriter := writer.NewInMemoryWriter()
	cfg := &axiom.Config{
		SubT: t,
		Case: &axiom.Case{
			ID:          "AUTH-1",
			Name:        "user can login",
			Description: "Valid credentials create a session.",
		},
		Meta: axiom.Meta{
			Feature:  "login",
			Severity: axiom.SeverityCritical,
		},
	}
	testallure.Plugin(allure.WithWriter(memoryWriter))(cfg)

	cfg.Runtime.Test(cfg, func(cfg *axiom.Config) {
		cfg.Setup("prepare user", func() {})
		cfg.Step("send request", func() {
			cfg.Runtime.Artefact(axiom.Artefact{
				Name: "request.json",
				Type: axiom.ArtefactTypeJSON,
				Data: []byte(`{"user":"alice"}`),
			})
			cfg.Step("validate status", func() {})
		})
		cfg.Teardown("remove user", func() {})
	})

	snapshot := memoryWriter.Snapshot()
	require.Len(t, snapshot.Results, 1)

	result := snapshot.Results[0]
	assert.Equal(t, "user can login", result.Name)
	assert.Equal(t, "Valid credentials create a session.", result.Description)
	assert.Equal(t, model.StatusPassed, result.Status)
	assertLabelValues(t, result.Labels, "ALLURE_ID", "AUTH-1")
	assertLabelValues(t, result.Labels, "feature", "login")
	assertLabelValues(t, result.Labels, "severity", "critical")

	require.Len(t, result.Steps, 3)
	assert.Equal(t, "prepare user", result.Steps[0].Name)
	assert.Equal(t, "send request", result.Steps[1].Name)
	assert.Equal(t, "remove user", result.Steps[2].Name)
	require.Len(t, result.Steps[1].Steps, 1)
	assert.Equal(t, "validate status", result.Steps[1].Steps[0].Name)

	require.Len(t, result.Steps[1].Attachments, 1)
	attachment := result.Steps[1].Attachments[0]
	assert.Equal(t, "request.json", attachment.Name)
	assert.Equal(t, "application/json", attachment.Type)
	assert.JSONEq(t, `{"user":"alice"}`, string(snapshot.Attachments[attachment.Source]))
}

func TestPlugin_ReportsFixtureCleanupArtefacts(t *testing.T) {
	memoryWriter := writer.NewInMemoryWriter()
	cfg := &axiom.Config{
		SubT: t,
		Case: &axiom.Case{
			Name: "fixture cleanup is reported",
		},
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"report": func(cfg *axiom.Config) (any, func(), error) {
					return struct{}{}, func() {
						cfg.Artefact(axiom.NewTextArtefact("final-report.txt", "finished"))
					}, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
	}
	testallure.Plugin(allure.WithWriter(memoryWriter))(cfg)

	cfg.Test(func(current *axiom.Config) {
		axiom.GetFixture[struct{}](current, "report")
	})

	snapshot := memoryWriter.Snapshot()
	require.Len(t, snapshot.Results, 1)
	require.Len(t, snapshot.Results[0].Attachments, 1)

	attachment := snapshot.Results[0].Attachments[0]
	assert.Equal(t, "final-report.txt", attachment.Name)
	assert.Equal(t, "text/plain", attachment.Type)
	assert.Equal(t, "finished", string(snapshot.Attachments[attachment.Source]))
}

func TestPlugin_ReportsFixtureCleanupAsTeardownStep(t *testing.T) {
	memoryWriter := writer.NewInMemoryWriter()
	cfg := &axiom.Config{
		SubT: t,
		Case: &axiom.Case{
			Name: "fixture cleanup teardown is reported",
		},
		Fixtures: axiom.Fixtures{
			Registry: map[string]axiom.Fixture{
				"report": func(cfg *axiom.Config) (any, func(), error) {
					return struct{}{}, func() {
						cfg.Teardown("finalize flow report", func() {
							cfg.Artefact(axiom.NewTextArtefact("flow-result.txt", "finished"))
						})
					}, nil
				},
			},
			Cache: map[string]axiom.FixtureResult{},
		},
	}
	testallure.Plugin(allure.WithWriter(memoryWriter))(cfg)

	cfg.Test(func(current *axiom.Config) {
		axiom.GetFixture[struct{}](current, "report")
	})

	snapshot := memoryWriter.Snapshot()
	require.Len(t, snapshot.Results, 1)
	require.Len(t, snapshot.Results[0].Steps, 1)

	teardown := snapshot.Results[0].Steps[0]
	assert.Equal(t, "finalize flow report", teardown.Name)
	assert.Equal(t, model.StatusPassed, teardown.Status)
	require.Len(t, teardown.Attachments, 1)

	attachment := teardown.Attachments[0]
	assert.Equal(t, "flow-result.txt", attachment.Name)
	assert.Equal(t, "text/plain", attachment.Type)
	assert.Equal(t, "finished", string(snapshot.Attachments[attachment.Source]))
}

func TestPlugin_ReportsSetupAndTeardownCalledFromTestHooks(t *testing.T) {
	memoryWriter := writer.NewInMemoryWriter()
	cfg := &axiom.Config{
		SubT: t,
		Case: &axiom.Case{
			Name: "hook lifecycle is reported",
		},
		Hooks: axiom.Hooks{
			BeforeTest: []axiom.TestHook{
				func(cfg *axiom.Config) {
					cfg.Setup("prepare hook state", func() {})
				},
			},
			AfterTest: []axiom.TestHook{
				func(cfg *axiom.Config) {
					cfg.Teardown("release hook state", func() {})
				},
			},
		},
	}
	testallure.Plugin(allure.WithWriter(memoryWriter))(cfg)

	cfg.Test(func(current *axiom.Config) {
		current.Step("test body", func() {})
	})

	snapshot := memoryWriter.Snapshot()
	require.Len(t, snapshot.Results, 1)
	require.Len(t, snapshot.Results[0].Steps, 3)
	assert.Equal(t, "prepare hook state", snapshot.Results[0].Steps[0].Name)
	assert.Equal(t, "test body", snapshot.Results[0].Steps[1].Name)
	assert.Equal(t, "release hook state", snapshot.Results[0].Steps[2].Name)
}

func TestPlugin_ParallelCasesKeepContextsIsolated(t *testing.T) {
	memoryWriter := writer.NewInMemoryWriter()
	var started sync.WaitGroup
	started.Add(2)

	runner := axiom.NewRunner(
		axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
		axiom.WithRunnerFixture("report", func(cfg *axiom.Config) (any, func(), error) {
			caseName := cfg.Case.Name
			return struct{}{}, func() {
				cfg.Artefact(axiom.NewTextArtefact(caseName+" cleanup.txt", caseName))
			}, nil
		}),
		axiom.WithRunnerPlugins(
			testallure.Plugin(allure.WithWriter(memoryWriter)),
		),
	)

	t.Run("parallel cases", func(t *testing.T) {
		for _, name := range []string{"alpha", "beta"} {
			name := name
			testCase := axiom.NewCase(axiom.WithCaseName(name))
			runner.RunCase(t, testCase, func(cfg *axiom.Config) {
				axiom.GetFixture[struct{}](cfg, "report")
				started.Done()
				started.Wait()

				cfg.Step(name+" step", func() {
					cfg.Runtime.Artefact(axiom.Artefact{
						Name: name + ".txt",
						Type: axiom.ArtefactTypeText,
						Data: []byte(name),
					})
				})
			})
		}
	})

	snapshot := memoryWriter.Snapshot()
	require.Len(t, snapshot.Results, 2)

	results := map[string]model.TestResult{}
	for _, result := range snapshot.Results {
		results[result.Name] = result
	}

	for _, name := range []string{"alpha", "beta"} {
		result, ok := results[name]
		require.True(t, ok, "missing result for %q", name)
		require.Len(t, result.Steps, 1)
		assert.Equal(t, name+" step", result.Steps[0].Name)
		require.Len(t, result.Steps[0].Attachments, 1)

		attachment := result.Steps[0].Attachments[0]
		assert.Equal(t, name+".txt", attachment.Name)
		assert.Equal(t, name, string(snapshot.Attachments[attachment.Source]))

		require.Len(t, result.Attachments, 1)
		cleanupAttachment := result.Attachments[0]
		assert.Equal(t, name+" cleanup.txt", cleanupAttachment.Name)
		assert.Equal(t, name, string(snapshot.Attachments[cleanupAttachment.Source]))
	}
}
