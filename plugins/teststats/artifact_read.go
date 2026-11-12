package teststats

import (
	"cmp"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

func readRunAttempts(dir, runID, processID string) ([]attemptArtifact, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var attempts []attemptArtifact
	for _, entry := range entries {
		if !isAttemptFile(entry.Name()) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		record, err := decodeAttemptArtifact(data)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		if record.RunID != runID || record.ProcessID != processID {
			return nil, fmt.Errorf("read %s: attempt belongs to a different run or process", path)
		}
		if entry.Name() != fmt.Sprintf("attempt-%d.json", record.Number) {
			return nil, fmt.Errorf("read %s: filename does not match attempt number %d", path, record.Number)
		}
		attempts = append(attempts, record)
	}
	slices.SortFunc(attempts, func(a, b attemptArtifact) int { return cmp.Compare(a.Number, b.Number) })
	return attempts, nil
}

func isAttemptFile(name string) bool {
	return strings.HasPrefix(name, "attempt-") && strings.HasSuffix(name, ".json")
}
