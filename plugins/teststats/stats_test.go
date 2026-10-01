package teststats

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStats_ZeroValueIsEmpty(t *testing.T) {
	var s Stats

	assert.Empty(t, s.Attempts())
	assert.Empty(t, s.Runs())
	assert.Zero(t, s.Summary())
}

func TestStats_AddAndQueriesCopyAttempts(t *testing.T) {
	s := NewStats()
	a := Attempt{RunID: "run", Status: StatusFailed, Meta: axiom.Meta{Tags: []string{"smoke"}}}
	s.add(a)
	a.Meta.Tags[0] = "changed by caller"

	got := s.Attempts()
	require.Len(t, got, 1)
	assert.Equal(t, []string{"smoke"}, got[0].Meta.Tags)
	got[0].Meta.Tags[0] = "changed by reader"
	assert.Equal(t, []string{"smoke"}, s.Attempts()[0].Meta.Tags)
}

func TestStats_AttemptsAreOrderedByStartRunAndNumber(t *testing.T) {
	start := time.Now()
	s := NewStats()
	for _, a := range []Attempt{
		{RunID: "b", Number: 1, Start: start},
		{RunID: "a", Number: 2, Start: start},
		{RunID: "a", Number: 1, Start: start},
		{RunID: "c", Number: 1, Start: start.Add(-time.Second)},
	} {
		s.add(a)
	}

	var order []string
	for _, a := range s.Attempts() {
		order = append(order, fmt.Sprintf("%s%d", a.RunID, a.Number))
	}
	assert.Equal(t, []string{"c1", "a1", "a2", "b1"}, order)
}

func TestStats_FilterSelectsCopiesAndMayReenter(t *testing.T) {
	s := NewStats()
	s.add(Attempt{RunID: "1", Status: StatusFailed, Meta: axiom.Meta{Tags: []string{"smoke"}}})
	s.add(Attempt{RunID: "2", Status: StatusPassed})
	s.add(Attempt{RunID: "3", Status: StatusSkipped})

	failed := s.Filter(func(a Attempt) bool {
		_ = s.Attempts()
		match := a.Status == StatusFailed
		if len(a.Meta.Tags) > 0 {
			a.Meta.Tags[0] = "changed by predicate"
		}
		return match
	})

	require.Len(t, failed.Attempts(), 1)
	assert.Equal(t, []string{"smoke"}, failed.Attempts()[0].Meta.Tags)
	assert.Equal(t, []string{"smoke"}, s.Attempts()[0].Meta.Tags)
	assert.Len(t, s.Attempts(), 3)
	assert.Empty(t, s.Filter(func(Attempt) bool { return false }).Attempts())
}

func TestStats_ConcurrentWritersAndReaders(t *testing.T) {
	s := NewStats()
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 20 {
				s.add(Attempt{Status: StatusPassed})
				_ = s.Attempts()
				_ = s.Filter(func(Attempt) bool { return true })
				_ = s.Runs()
				_ = s.Summary()
			}
		})
	}
	wg.Wait()

	assert.Len(t, s.Attempts(), 160)
}
