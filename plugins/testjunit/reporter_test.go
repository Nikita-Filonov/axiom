package testjunit

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReporterSnapshotAndLateFailure(t *testing.T) {
	r := NewReporter()
	start := time.Now()
	late := r.add(attempt{TestName: "later", Start: start.Add(time.Second)})
	r.add(attempt{TestName: "b", Start: start})
	r.add(attempt{TestName: "a", Start: start})
	first := r.snapshot()
	require.Len(t, first, 3)
	assert.Equal(t, "a", first[0].TestName)
	assert.Equal(t, "b", first[1].TestName)
	assert.Equal(t, "later", first[2].TestName)
	first[0].TestName = "changed"
	r.markFailed(late)
	second := r.snapshot()
	require.Len(t, second, 3)
	assert.Equal(t, "a", second[0].TestName, "snapshots share data")
	assert.Equal(t, statusFailed, second[2].Status, "missed late failure")
	assert.Equal(t, statusPassed, first[2].Status, "snapshots share data")
}

func TestReporterZeroValueAndIndependentInstallation(t *testing.T) {
	var first, second Reporter
	var output bytes.Buffer
	require.NoError(t, first.Write(&output))
	var doc xmlSuites
	require.NoError(t, xml.Unmarshal(output.Bytes(), &doc))
	assert.Zero(t, doc.Tests)
	assert.Empty(t, doc.Suites)
	t.Run("attempt", func(t *testing.T) {
		cfg := &axiom.Config{SubT: t}
		Plugin(&first)(cfg)
		Plugin(&first)(cfg)
		Plugin(&second, WithSuiteName("second"))(cfg)
		assert.Len(t, cfg.Runtime.EventSinks, 2)
		cfg.Event(axiom.Event{Type: axiom.EventTypeCaseStart})
	})
	firstResults, secondResults := first.snapshot(), second.snapshot()
	require.Len(t, firstResults, 1)
	require.Len(t, secondResults, 1)
	assert.Equal(t, "axiom", firstResults[0].SuiteName)
	assert.Equal(t, "second", secondResults[0].SuiteName)
}

func TestReporterConcurrentSnapshots(t *testing.T) {
	r := NewReporter()
	var workers sync.WaitGroup
	for i := range 32 {
		workers.Go(func() {
			result := r.add(attempt{TestName: fmt.Sprintf("attempt-%02d", i)})
			r.markFailed(result)
			var output bytes.Buffer
			if !assert.NoError(t, r.Write(&output)) {
				return
			}
			var doc xmlSuites
			if !assert.NoError(t, xml.Unmarshal(output.Bytes(), &doc)) || !assert.Len(t, doc.Suites, 1) {
				return
			}
			assert.Equal(t, len(doc.Suites[0].Cases), doc.Tests, "inconsistent snapshot")
		})
	}
	workers.Wait()
	assert.Len(t, r.snapshot(), 32, "concurrent collection lost attempts")
}
