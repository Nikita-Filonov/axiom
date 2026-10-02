package testjunit_test

import (
	"encoding/xml"
	"os"
	"path/filepath"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testjunit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJUnitReportExample(t *testing.T) {
	// The reporter stores results; Plugin configures how attempts are recorded.
	reporter := testjunit.NewReporter()
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testjunit.Plugin(reporter, testjunit.WithSuiteName("example/package")),
		),
	)

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("request succeeds")), func(cfg *axiom.Config) {
		// Test the application here. Failures still affect go test normally.
		cfg.T().Log("request succeeded")
	})

	// Sequential RunCase calls already have a result to export.
	// To include late cleanup failures and parallel cases, export after
	// the parent test or m.Run finishes.
	path := filepath.Join(t.TempDir(), "junit.xml")
	require.NoError(t, reporter.WriteFile(path))
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	var report struct {
		Tests int `xml:"tests,attr"`
	}
	require.NoError(t, xml.Unmarshal(data, &report))
	assert.Equal(t, 1, report.Tests)
}
