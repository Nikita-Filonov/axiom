package testjunit_test

import (
	"bytes"
	"encoding/xml"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testjunit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type reportXML struct {
	Tests    int `xml:"tests,attr"`
	Failures int `xml:"failures,attr"`
	Skipped  int `xml:"skipped,attr"`
	Suites   []struct {
		Name  string `xml:"name,attr"`
		Cases []struct {
			Name      string `xml:"name,attr"`
			ClassName string `xml:"classname,attr"`
			Failure   *struct {
				Message string `xml:"message,attr"`
			} `xml:"failure"`
			Skipped *struct {
				Message string `xml:"message,attr"`
			} `xml:"skipped"`
		} `xml:"testcase"`
	} `xml:"testsuite"`
}

func TestReporterPlugin(t *testing.T) {
	reporter := testjunit.NewReporter()
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(
		testjunit.Plugin(reporter, testjunit.WithSuiteName("example/package")),
		testjunit.Plugin(reporter, testjunit.WithSuiteName("ignored duplicate")),
	))
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("passes")), func(*axiom.Config) {})
	runner.RunCase(t, axiom.NewCase(
		axiom.WithCaseName("skipped"),
		axiom.WithCaseSkip(axiom.SkipBecause("feature disabled")),
	), func(cfg *axiom.Config) { assert.Fail(cfg.T(), "skipped body ran") })

	var output bytes.Buffer
	require.NoError(t, reporter.Write(&output))
	var report reportXML
	require.NoError(t, xml.Unmarshal(output.Bytes(), &report))
	assert.Equal(t, 2, report.Tests)
	assert.Zero(t, report.Failures)
	assert.Equal(t, 1, report.Skipped)
	require.Len(t, report.Suites, 1)
	suite := report.Suites[0]
	assert.Equal(t, "example/package", suite.Name)
	require.Len(t, suite.Cases, 2)
	assert.Contains(t, suite.Cases[0].Name, "passes")
	require.NotNil(t, suite.Cases[1].Skipped)
	assert.Equal(t, "feature disabled", suite.Cases[1].Skipped.Message)
}
