package testjunit

import (
	"bytes"
	"encoding/xml"
	"errors"
	"math"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDocument(t *testing.T) {
	doc := (reportSnapshot{
		{SuiteName: "service & API", TestName: "TestPass", Status: statusPassed, Duration: 1500 * time.Millisecond},
		{SuiteName: "service & API", TestName: "TestFail", Status: statusFailed, Error: "step <failed>", Duration: 250 * time.Millisecond},
		{SuiteName: "service & API", TestName: "TestError", Status: statusFailed},
		{SuiteName: "service & API", TestName: "TestSkip", Status: statusSkipped, SkipReason: "disabled & flaky"},
		{SuiteName: "service & API", TestName: "TestSkipEmpty", Status: statusSkipped, Duration: -time.Second},
	}).document()
	assert.Equal(t, 5, doc.Tests)
	assert.Equal(t, 2, doc.Failures)
	assert.Equal(t, 2, doc.Skipped)
	assert.Equal(t, "1.750000", doc.Time)
	require.Len(t, doc.Suites, 1)
	suite := doc.Suites[0]
	assert.Equal(t, "service & API", suite.Name)
	assert.Equal(t, 5, suite.Tests)
	assert.Equal(t, 2, suite.Failures)
	assert.Equal(t, 2, suite.Skipped)
	assert.Equal(t, doc.Time, suite.Time)
	require.Len(t, suite.Cases, 5)
	assert.Equal(t, "1.500000", suite.Cases[0].Time)
	assert.Nil(t, suite.Cases[0].Failure)
	assert.Nil(t, suite.Cases[0].Skipped)
	require.NotNil(t, suite.Cases[1].Failure)
	assert.Equal(t, "step <failed>", suite.Cases[1].Failure.Message)
	require.NotNil(t, suite.Cases[2].Failure)
	assert.Equal(t, "test failed", suite.Cases[2].Failure.Message)
	require.NotNil(t, suite.Cases[3].Skipped)
	assert.Equal(t, "disabled & flaky", suite.Cases[3].Skipped.Message)
	require.NotNil(t, suite.Cases[4].Skipped)
	assert.Equal(t, "skipped", suite.Cases[4].Skipped.Message)
	data, err := xml.Marshal(doc)
	require.NoError(t, err)
	assert.Contains(t, string(data), "service &amp; API")
	assert.Contains(t, string(data), "step &lt;failed&gt;")
}

func TestCleanXML(t *testing.T) {
	got := cleanXML("a\x00\x01\t\n\r&\xff\U0001f600")
	assert.Equal(t, "a��\t\n\r&�😀", got)
	doc := (reportSnapshot{{
		SuiteName: "bad\x00name", TestName: "case\x01", Status: statusFailed, Error: "bad\x00message",
	}}).document()
	_, err := xml.Marshal(doc)
	require.NoError(t, err)
}

func TestDocumentMultipleSuites(t *testing.T) {
	doc := (reportSnapshot{
		{SuiteName: "second", TestName: "fail", Status: statusFailed, Duration: time.Second},
		{SuiteName: "first", TestName: "skip", Status: statusSkipped, Duration: 2 * time.Second},
		{SuiteName: "second", TestName: "pass", Duration: 3 * time.Second},
		{TestName: "default", Duration: -time.Second},
		{SuiteName: "axiom", TestName: "explicit default"},
	}).document()
	assert.Equal(t, 5, doc.Tests)
	assert.Equal(t, 1, doc.Failures)
	assert.Equal(t, 1, doc.Skipped)
	assert.Equal(t, "6.000000", doc.Time)
	require.Len(t, doc.Suites, 3)
	for i, want := range []struct {
		name     string
		tests    int
		failures int
		skipped  int
		time     string
		names    []string
	}{
		{"second", 2, 1, 0, "4.000000", []string{"fail", "pass"}},
		{"first", 1, 0, 1, "2.000000", []string{"skip"}},
		{"axiom", 2, 0, 0, "0.000000", []string{"default", "explicit default"}},
	} {
		suite := doc.Suites[i]
		assert.Equal(t, want.name, suite.Name)
		assert.Equal(t, want.tests, suite.Tests)
		assert.Equal(t, want.failures, suite.Failures)
		assert.Equal(t, want.skipped, suite.Skipped)
		assert.Equal(t, want.time, suite.Time)
		require.Len(t, suite.Cases, want.tests)
		for j, result := range suite.Cases {
			assert.Equal(t, want.name, result.ClassName)
			assert.Equal(t, want.names[j], result.Name)
		}
	}
}

func TestDocumentSanitizedSuiteNames(t *testing.T) {
	doc := (reportSnapshot{
		{SuiteName: "invalid\x00", TestName: "one"},
		{SuiteName: "invalid\xff", TestName: "two"},
		{SuiteName: "invalid�", TestName: "three"},
	}).document()
	require.Len(t, doc.Suites, 1)
	assert.Equal(t, "invalid�", doc.Suites[0].Name)
	assert.Equal(t, 3, doc.Suites[0].Tests)
}

func TestDocumentLargeDurationAndXMLBoundaries(t *testing.T) {
	name := "quotes \"' & < >\t\n\r\ud7ff\ue000\ufffd😀"
	errText := "failure ]]>\x00\x08\x0b\x0c\x0e\x1f\ufffe\uffff"
	doc := (reportSnapshot{
		{SuiteName: name, TestName: name, Status: statusFailed, Error: errText, Duration: time.Duration(math.MaxInt64)},
		{SuiteName: name, TestName: "second", Duration: time.Duration(math.MaxInt64)},
	}).document()
	total, err := strconv.ParseFloat(doc.Time, 64)
	require.NoError(t, err)
	assert.Greater(t, total, time.Duration(math.MaxInt64).Seconds(), "overflowed total time")
	data, err := xml.Marshal(doc)
	require.NoError(t, err)
	var decoded xmlSuites
	require.NoError(t, xml.Unmarshal(data, &decoded))
	require.Len(t, decoded.Suites, 1)
	suite := decoded.Suites[0]
	assert.Equal(t, name, suite.Name)
	require.Len(t, suite.Cases, 2)
	assert.Equal(t, name, suite.Cases[0].Name)
	require.NotNil(t, suite.Cases[0].Failure)
	assert.Equal(t, "failure ]]>��������", suite.Cases[0].Failure.Text)
}

type failingWriter struct{ err error }

func (w failingWriter) Write([]byte) (int, error) { return 0, w.err }

func TestWrite(t *testing.T) {
	r := NewReporter()
	var output bytes.Buffer
	require.NoError(t, r.Write(&output))
	assert.True(t, strings.HasPrefix(output.String(), xml.Header), "missing XML header")
	assert.True(t, strings.HasSuffix(output.String(), "</testsuites>\n"), "missing trailing newline")
	var parsed xmlSuites
	require.NoError(t, xml.Unmarshal(output.Bytes(), &parsed))
	assert.Zero(t, parsed.Tests)
	assert.Zero(t, parsed.Failures)
	assert.Zero(t, parsed.Skipped)
	assert.Equal(t, "0.000000", parsed.Time)
	assert.Empty(t, parsed.Suites)
	require.Error(t, r.Write(nil), "nil writer accepted")
	var nilReporter *Reporter
	require.Error(t, nilReporter.Write(&output), "nil reporter accepted")
	want := errors.New("write failed")
	require.ErrorIs(t, r.Write(failingWriter{want}), want)
}
