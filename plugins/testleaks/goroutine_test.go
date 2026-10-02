package testleaks

import (
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/google/pprof/profile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadGoroutinesErrors(t *testing.T) {
	want := errors.New("profile unavailable")
	_, err := readGoroutines("attempt", nil, func(io.Writer) error { return want })
	require.ErrorIs(t, err, want)
	_, err = readGoroutines("attempt", nil, func(w io.Writer) error {
		_, err := io.WriteString(w, "not a profile")
		return err
	})
	require.Error(t, err, "malformed profile parsed successfully")
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
	require.Len(t, got, 2)
	assert.Equal(t, int64(4), goroutineCount(got))
	assert.Contains(t, got[0].stack, "worker.go:12")
	assert.Contains(t, got[1].stack, "other.go:34")
}

func TestFormatStackCornerCases(t *testing.T) {
	assert.Empty(t, formatStack(&profile.Sample{}))
	lines := []profile.Line{{}}
	for i := 0; i < 17; i++ {
		lines = append(lines, profile.Line{Function: &profile.Function{Name: "frame", Filename: "f.go"}, Line: int64(i + 1)})
	}
	got := formatStack(&profile.Sample{Location: []*profile.Location{{Line: lines}}})
	assert.Equal(t, 16, strings.Count(got, "  frame\n"))
	assert.True(t, strings.HasSuffix(got, "  ..."), "stack truncation = %q", got)
}
