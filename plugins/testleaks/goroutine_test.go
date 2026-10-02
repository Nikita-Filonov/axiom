package testleaks

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/pprof/profile"
)

func TestReadGoroutinesErrors(t *testing.T) {
	want := errors.New("profile unavailable")
	_, err := readGoroutines("attempt", nil, func(io.Writer) error { return want })
	if !errors.Is(err, want) {
		t.Fatalf("capture error = %v, want %v", err, want)
	}
	_, err = readGoroutines("attempt", nil, func(w io.Writer) error {
		_, err := io.WriteString(w, "not a profile")
		return err
	})
	if err == nil {
		t.Fatal("malformed profile parsed successfully")
	}
}

func TestReadGoroutinesSamples(t *testing.T) {
	matching := &profile.Sample{
		Label: map[string][]string{labelKey: {"other", "attempt"}},
		Value: []int64{3},
		Location: []*profile.Location{{ID: 1, Line: []profile.Line{{
			Function: &profile.Function{ID: 1, Name: "worker", Filename: "worker.go"}, Line: 12,
		}}}},
	}
	withoutValue := &profile.Sample{
		Label: map[string][]string{labelKey: {"attempt"}},
		Location: []*profile.Location{{ID: 2, Line: []profile.Line{{
			Function: &profile.Function{ID: 2, Name: "otherWorker", Filename: "other.go"}, Line: 34,
		}}}},
	}
	wrongAttempt := &profile.Sample{
		Label: map[string][]string{labelKey: {"another"}},
		Location: []*profile.Location{{ID: 4, Line: []profile.Line{{
			Function: &profile.Function{ID: 4, Name: "unrelated"},
		}}}},
	}
	ignored := &profile.Sample{
		Label: map[string][]string{labelKey: {"attempt"}},
		Location: []*profile.Location{{ID: 3, Line: []profile.Line{{
			Function: &profile.Function{ID: 3, Name: "keepAlive"},
		}}}},
	}
	p := &profile.Profile{Sample: []*profile.Sample{matching, withoutValue, wrongAttempt, ignored}}
	got := goroutinesFromProfile(p, "attempt", []string{"keepAlive"})
	if len(got) != 2 || goroutineCount(got) != 4 {
		t.Fatalf("goroutines = %+v, want two groups and four workers", got)
	}
	if !strings.Contains(got[0].stack, "worker.go:12") || !strings.Contains(got[1].stack, "other.go:34") {
		t.Fatalf("unexpected stacks: %+v", got)
	}
}

func TestFormatStackCornerCases(t *testing.T) {
	if got := formatStack(&profile.Sample{}); got != "" {
		t.Fatalf("empty stack = %q", got)
	}
	lines := []profile.Line{{}}
	for i := 0; i < 17; i++ {
		lines = append(lines, profile.Line{Function: &profile.Function{Name: "frame", Filename: "f.go"}, Line: int64(i + 1)})
	}
	got := formatStack(&profile.Sample{Location: []*profile.Location{{Line: lines}}})
	if strings.Count(got, "  frame\n") != 16 || !strings.HasSuffix(got, "  ...") {
		t.Fatalf("stack truncation = %q", got)
	}
}
