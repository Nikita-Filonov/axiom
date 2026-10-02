package testjunit_test

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testjunit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func readReport(t *testing.T, reporter *testjunit.Reporter) reportXML {
	t.Helper()
	var output bytes.Buffer
	require.NoError(t, reporter.Write(&output))
	var report reportXML
	require.NoError(t, xml.Unmarshal(output.Bytes(), &report))
	return report
}

func TestParallelCasesAndPolicySkip(t *testing.T) {
	reporter := testjunit.NewReporter()
	t.Run("suite", func(t *testing.T) {
		runner := axiom.NewRunner(
			axiom.WithRunnerPlugins(testjunit.Plugin(reporter)),
			axiom.WithRunnerParallel(axiom.WithParallelEnabled()),
			axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
		)
		for i := range 12 {
			runner.RunCase(t, axiom.NewCase(axiom.WithCaseName(fmt.Sprintf("case-%d", i))), func(cfg *axiom.Config) {
				// Concurrent exports may see earlier attempts, but never a partial record.
				report := readReport(cfg.T(), reporter)
				var cases int
				for _, suite := range report.Suites {
					cases += len(suite.Cases)
				}
				assert.Equal(cfg.T(), cases, report.Tests, "inconsistent snapshot")
			})
		}
		runner.RunCase(t, axiom.NewCase(
			axiom.WithCaseName("policy skip"),
			axiom.WithCaseSkip(axiom.SkipBecause("disabled")),
		), func(cfg *axiom.Config) { assert.Fail(cfg.T(), "policy-skipped body ran") })
	})
	report := readReport(t, reporter)
	assert.Equal(t, 13, report.Tests)
	assert.Zero(t, report.Failures)
	assert.Equal(t, 1, report.Skipped)
}

func TestDirectSkip(t *testing.T) {
	reporter := testjunit.NewReporter()
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(testjunit.Plugin(reporter)))
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("direct skip")), func(cfg *axiom.Config) { cfg.T().Skip("not exposed by Go") })
	report := readReport(t, reporter)
	assert.Equal(t, 1, report.Tests)
	assert.Equal(t, 1, report.Skipped)
	require.Len(t, report.Suites, 1)
	require.Len(t, report.Suites[0].Cases, 1)
	require.NotNil(t, report.Suites[0].Cases[0].Skipped)
	assert.Equal(t, "skipped", report.Suites[0].Cases[0].Skipped.Message)
}

func TestSharedReporterAndPluginOptions(t *testing.T) {
	reporter := testjunit.NewReporter()
	installations := []struct {
		name   string
		plugin axiom.Plugin
	}{
		{"axiom", testjunit.Plugin(reporter)},
		{"service-a", testjunit.Plugin(reporter, testjunit.WithSuiteName("service-a"))},
		{"service-b", testjunit.Plugin(reporter, testjunit.WithSuiteName("service-b"))},
	}
	t.Run("runners", func(t *testing.T) {
		for i := range 6 {
			installation := installations[i%len(installations)]
			t.Run(fmt.Sprintf("%s-%d", installation.name, i), func(t *testing.T) {
				t.Parallel()
				// The same plugin can be reused by separate runners concurrently.
				runner := axiom.NewRunner(axiom.WithRunnerPlugins(installation.plugin))
				for _, name := range []string{"first", "second"} {
					runner.RunCase(t, axiom.NewCase(
						axiom.WithCaseName(name),
						// Reinstalling at case level must neither duplicate nor rename results.
						axiom.WithCasePlugins(testjunit.Plugin(reporter, testjunit.WithSuiteName("ignored"))),
					), func(*axiom.Config) {})
				}
			})
		}
	})
	report := readReport(t, reporter)
	assert.Equal(t, 12, report.Tests)
	assert.Zero(t, report.Failures)
	assert.Zero(t, report.Skipped)
	assert.Len(t, report.Suites, 3)
	seen := make(map[string]bool)
	for _, suite := range report.Suites {
		assert.NotContains(t, seen, suite.Name, "suite was duplicated")
		assert.Len(t, suite.Cases, 4, "suite lost cases")
		seen[suite.Name] = true
		for _, result := range suite.Cases {
			assert.Equal(t, suite.Name, result.ClassName)
			assert.Contains(t, result.Name, "/"+suite.Name+"-", "attempt assigned to wrong suite")
		}
	}
	for _, installation := range installations {
		assert.Contains(t, seen, installation.name, "missing suite")
	}
}

func TestSelectionExcludesUnrunCases(t *testing.T) {
	exe, err := os.Executable()
	require.NoError(t, err)
	cmd := exec.Command(exe, "-test.run=^TestSelectionProbe$/^included$")
	cmd.Env = append(os.Environ(), "AXIOM_JUNIT_SELECTION=1")
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "selection probe: %s", output)
}

func TestSelectionProbe(t *testing.T) {
	if os.Getenv("AXIOM_JUNIT_SELECTION") == "" {
		return
	}
	reporter := testjunit.NewReporter()
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(testjunit.Plugin(reporter)))
	for _, name := range []string{"included", "excluded"} {
		runner.RunCase(t, axiom.NewCase(axiom.WithCaseName(name)), func(*axiom.Config) {})
	}
	assert.Equal(t, 1, readReport(t, reporter).Tests, "excluded case recorded")
}
