package testleaks

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordingReporter struct {
	helpers  int
	messages []string
}

func (r *recordingReporter) Helper() { r.helpers++ }
func (r *recordingReporter) Errorf(format string, args ...any) {
	r.messages = append(r.messages, fmt.Sprintf(format, args...))
}

func TestVerify(t *testing.T) {
	t.Run("clean without goroutine sampling", func(t *testing.T) {
		r := &recordingReporter{}
		verify(r, newAttemptState(), "id", Config{}, func(string, []string) ([]goroutine, error) {
			assert.Fail(t, "goroutine finder called while disabled")
			return nil, nil
		})
		assert.Equal(t, 1, r.helpers)
		assert.Empty(t, r.messages)
	})
	t.Run("profile error", func(t *testing.T) {
		r := &recordingReporter{}
		verify(r, newAttemptState(), "id", Config{Goroutines: true, IgnoreFunctions: []string{"intentional"}}, func(id string, ignored []string) ([]goroutine, error) {
			assert.Equal(t, "id", id)
			assert.Equal(t, []string{"intentional"}, ignored)
			return nil, errors.New("broken profile")
		})
		assert.Equal(t, []string{"testleaks: goroutine profile: broken profile"}, r.messages)
	})
	t.Run("worker exits during grace period", func(t *testing.T) {
		r := &recordingReporter{}
		calls := 0
		verify(r, newAttemptState(), "id", Config{Goroutines: true, GracePeriod: time.Second}, func(string, []string) ([]goroutine, error) {
			calls++
			if calls == 1 {
				return []goroutine{{count: 1}}, nil
			}
			return nil, nil
		})
		assert.Equal(t, 2, calls)
		assert.Empty(t, r.messages)
	})
	t.Run("resource remains after grace period", func(t *testing.T) {
		r := &recordingReporter{}
		state := newAttemptState()
		state.add("socket", "socket.go:12")
		verify(r, state, "id", Config{GracePeriod: time.Millisecond}, nil)
		require.Len(t, r.messages, 1)
		assert.Contains(t, r.messages[0], "socket (registered at socket.go:12)")
	})
}

func TestFormatReport(t *testing.T) {
	assert.Empty(t, formatReport(nil, nil))
	groups := make([]goroutine, 9)
	for i := range groups {
		groups[i] = goroutine{count: 2, stack: "worker stack"}
	}
	got := formatReport(groups, []trackedResource{{name: "db", site: "db.go:9"}})
	for _, want := range []string{
		"18 goroutine(s) still running", "... 1 more stack group(s)",
		"1 tracked resource(s) not released", "- db (registered at db.go:9)",
	} {
		assert.Contains(t, got, want)
	}
	assert.Equal(t, 8, strings.Count(got, "worker stack"))
}
