package teststats

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (l *resultLoader) loadRun(processID, runKey string) {
	runPath := path.Join("processes", processID, "runs", runKey)
	runID, err := decodeRunKey(runKey)
	if err != nil {
		l.issue(runPath, err)
		return
	}
	entries, err := fs.ReadDir(l.files, runPath)
	if err != nil {
		l.issue(runPath, err)
		return
	}
	var numbers []int
	found := false
	for _, entry := range entries {
		if !isAttemptFile(entry.Name()) {
			continue
		}
		found = true
		filePath := path.Join(runPath, entry.Name())
		number, err := parseAttemptNumber(entry.Name())
		if err != nil {
			l.issue(filePath, err)
			continue
		}
		numbers = append(numbers, number)
		record, err := l.readAttempt(runPath, entry)
		if err != nil {
			l.issue(filePath, err)
			continue
		}
		if record.ProcessID != processID || record.RunID != runID || record.Number != number {
			l.issue(filePath, errors.New("attempt identity does not match its path"))
			continue
		}
		l.addAttempt(filePath, record)
	}
	if !found {
		l.issue(runPath, errors.New("no attempt files"))
		return
	}
	l.checkMissingAttempts(runPath, numbers)
}

func decodeRunKey(key string) (string, error) {
	decoded, err := hex.DecodeString(key)
	if err != nil || len(decoded) == 0 || !utf8.Valid(decoded) || hex.EncodeToString(decoded) != key {
		return "", errors.New("run directory must be a lowercase hex-encoded run ID")
	}
	return string(decoded), nil
}

func parseAttemptNumber(name string) (int, error) {
	value := strings.TrimSuffix(strings.TrimPrefix(name, "attempt-"), ".json")
	number, err := strconv.Atoi(value)
	if err != nil || number < 1 || name != fmt.Sprintf("attempt-%d.json", number) {
		return 0, errors.New("expected attempt-<positive integer>.json without leading zeros")
	}
	return number, nil
}

func (l *resultLoader) readAttempt(dir string, entry fs.DirEntry) (attemptArtifact, error) {
	if !entry.Type().IsRegular() {
		return attemptArtifact{}, errors.New("expected a regular attempt file")
	}
	data, err := fs.ReadFile(l.files, path.Join(dir, entry.Name()))
	if err != nil {
		return attemptArtifact{}, err
	}
	return decodeAttemptArtifact(data)
}

func (l *resultLoader) addAttempt(path string, record attemptArtifact) {
	attempt := record.toAttempt()
	if record.State != StateRunning {
		l.result.Stats.add(attempt)
	}
	if record.State != StateFinalized {
		l.result.Unfinished = append(l.result.Unfinished, UnfinishedAttempt{
			Path:    path,
			State:   record.State,
			Attempt: attempt,
		})
	}
}

func (l *resultLoader) checkMissingAttempts(runPath string, numbers []int) {
	slices.Sort(numbers)
	previous := 0
	for _, number := range numbers {
		if number-previous > 1 {
			l.issue(runPath, fmt.Errorf("missing attempts %d through %d", previous+1, number-1))
		}
		previous = number
	}
}
