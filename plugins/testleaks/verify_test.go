package testleaks

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
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
			t.Fatal("goroutine finder called while disabled")
			return nil, nil
		})
		if r.helpers != 1 || len(r.messages) != 0 {
			t.Fatalf("reporter = %+v", r)
		}
	})
	t.Run("profile error", func(t *testing.T) {
		r := &recordingReporter{}
		verify(r, newAttemptState(), "id", Config{Goroutines: true, IgnoreFunctions: []string{"intentional"}}, func(id string, ignored []string) ([]goroutine, error) {
			if id != "id" || len(ignored) != 1 || ignored[0] != "intentional" {
				t.Fatalf("finder arguments: %q, %v", id, ignored)
			}
			return nil, errors.New("broken profile")
		})
		if len(r.messages) != 1 || r.messages[0] != "testleaks: goroutine profile: broken profile" {
			t.Fatalf("messages = %v", r.messages)
		}
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
		if calls != 2 || len(r.messages) != 0 {
			t.Fatalf("calls = %d, messages = %v", calls, r.messages)
		}
	})
	t.Run("resource remains after grace period", func(t *testing.T) {
		r := &recordingReporter{}
		state := newAttemptState()
		state.add("socket", "socket.go:12")
		verify(r, state, "id", Config{GracePeriod: time.Millisecond}, nil)
		if len(r.messages) != 1 || !strings.Contains(r.messages[0], "socket (registered at socket.go:12)") {
			t.Fatalf("messages = %v", r.messages)
		}
	})
}

func TestFormatReport(t *testing.T) {
	if got := formatReport(nil, nil); got != "" {
		t.Fatalf("empty report = %q", got)
	}
	groups := make([]goroutine, 9)
	for i := range groups {
		groups[i] = goroutine{count: 2, stack: "worker stack"}
	}
	got := formatReport(groups, []trackedResource{{name: "db", site: "db.go:9"}})
	for _, want := range []string{
		"18 goroutine(s) still running", "... 1 more stack group(s)",
		"1 tracked resource(s) not released", "- db (registered at db.go:9)",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("report missing %q: %s", want, got)
		}
	}
	if strings.Count(got, "worker stack") != 8 {
		t.Fatalf("reported stack groups = %d, want 8", strings.Count(got, "worker stack"))
	}
}
