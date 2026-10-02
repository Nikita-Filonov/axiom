package testleaks

import (
	"bytes"
	"fmt"
	"io"
	"runtime/pprof"
	"strings"

	"github.com/google/pprof/profile"
)

type goroutine struct {
	count int64
	stack string
}

func findGoroutines(id string, ignored []string) ([]goroutine, error) {
	return readGoroutines(id, ignored, func(w io.Writer) error {
		return pprof.Lookup("goroutine").WriteTo(w, 0)
	})
}

func readGoroutines(id string, ignored []string, write func(io.Writer) error) ([]goroutine, error) {
	var data bytes.Buffer
	if err := write(&data); err != nil {
		return nil, err
	}
	parsed, err := profile.Parse(&data)
	if err != nil {
		return nil, err
	}
	return goroutinesFromProfile(parsed, id, ignored), nil
}

func goroutinesFromProfile(parsed *profile.Profile, id string, ignored []string) []goroutine {
	var found []goroutine
	for _, sample := range parsed.Sample {
		if !hasLabel(sample.Label[labelKey], id) || ignoredStack(sample, ignored) {
			continue
		}
		count := int64(1)
		if len(sample.Value) > 0 {
			count = sample.Value[0]
		}
		found = append(found, goroutine{count: count, stack: formatStack(sample)})
	}
	return found
}

func hasLabel(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func ignoredStack(sample *profile.Sample, ignored []string) bool {
	for _, location := range sample.Location {
		for _, line := range location.Line {
			for _, name := range ignored {
				if line.Function != nil && line.Function.Name == name {
					return true
				}
			}
		}
	}
	return false
}

func formatStack(sample *profile.Sample) string {
	var out []byte
	frames := 0
	for _, location := range sample.Location {
		for _, line := range location.Line {
			if line.Function == nil {
				continue
			}
			if frames >= 16 {
				out = append(out, "  ...\n"...)
				return strings.TrimRight(string(out), "\n")
			}
			out = fmt.Appendf(out, "  %s\n    %s:%d\n", line.Function.Name, line.Function.Filename, line.Line)
			frames++
		}
	}
	return strings.TrimRight(string(out), "\n")
}

func goroutineCount(goroutines []goroutine) int64 {
	var count int64
	for _, g := range goroutines {
		count += g.count
	}
	return count
}
