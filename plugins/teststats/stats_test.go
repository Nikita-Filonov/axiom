package teststats_test

import (
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
	"github.com/stretchr/testify/assert"
)

func newCR(status string) *teststats.CaseResult {
	return &teststats.CaseResult{
		ID:     "1",
		Name:   "case",
		Status: status,
		Meta:   axiom.Meta{},
	}
}

func TestStats_Record_Passed(t *testing.T) {
	s := teststats.NewStats()

	cr := newCR(teststats.StatusPassed)
	s.Record(cr)

	assert.Equal(t, 1, s.Total)
	assert.Equal(t, 1, s.Passed)
	assert.Equal(t, 0, s.Failed)
	assert.Equal(t, 0, s.Skipped)
	assert.Equal(t, 0, s.Flaky)
	assert.Len(t, s.Cases, 1)
}

func TestStats_Record_Failed(t *testing.T) {
	s := teststats.NewStats()

	cr := newCR(teststats.StatusFailed)
	s.Record(cr)

	assert.Equal(t, 1, s.Total)
	assert.Equal(t, 0, s.Passed)
	assert.Equal(t, 1, s.Failed)
	assert.Equal(t, 0, s.Skipped)
	assert.Equal(t, 0, s.Flaky)
	assert.Len(t, s.Cases, 1)
}

func TestStats_Record_Skipped(t *testing.T) {
	s := teststats.NewStats()

	cr := newCR(teststats.StatusSkipped)
	s.Record(cr)

	assert.Equal(t, 1, s.Total)
	assert.Equal(t, 0, s.Passed)
	assert.Equal(t, 0, s.Failed)
	assert.Equal(t, 1, s.Skipped)
	assert.Equal(t, 0, s.Flaky)
	assert.Len(t, s.Cases, 1)
}

func TestStats_Record_Flaky(t *testing.T) {
	s := teststats.NewStats()

	cr := newCR(teststats.StatusFlaky)
	s.Record(cr)

	assert.Equal(t, 1, s.Total)
	assert.Equal(t, 0, s.Passed)
	assert.Equal(t, 0, s.Failed)
	assert.Equal(t, 0, s.Skipped)
	assert.Equal(t, 1, s.Flaky)
	assert.Len(t, s.Cases, 1)
}

func TestStats_Record_MultipleCases(t *testing.T) {
	s := teststats.NewStats()

	s.Record(newCR(teststats.StatusPassed))
	s.Record(newCR(teststats.StatusFailed))
	s.Record(newCR(teststats.StatusSkipped))
	s.Record(newCR(teststats.StatusFlaky))
	s.Record(newCR(teststats.StatusPassed))

	assert.Equal(t, 5, s.Total)
	assert.Equal(t, 2, s.Passed)
	assert.Equal(t, 1, s.Failed)
	assert.Equal(t, 1, s.Skipped)
	assert.Equal(t, 1, s.Flaky)
	assert.Len(t, s.Cases, 5)
}

func TestStats_Record_Concurrent(t *testing.T) {
	s := teststats.NewStats()

	wg := sync.WaitGroup{}
	wg.Add(100)

	for i := 0; i < 100; i++ {
		go func() {
			s.Record(newCR(teststats.StatusPassed))
			wg.Done()
		}()
	}

	wg.Wait()

	assert.Equal(t, 100, s.Total)
	assert.Equal(t, 100, s.Passed)
	assert.Len(t, s.Cases, 100)
}
