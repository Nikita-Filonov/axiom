package teststats

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sync"
)

type outputWriter struct {
	dir     string
	onError func(error)
	mu      sync.Mutex
}

func newOutputWriter(dir string, onError func(error)) *outputWriter {
	if dir == "" {
		return nil
	}
	return &outputWriter{
		dir:     dir,
		onError: onError,
	}
}

func (w *outputWriter) publish(record attemptArtifact) {
	err := w.write(record)
	if err != nil {
		w.onError(fmt.Errorf("teststats: export run %q attempt %d (%s): %w", record.RunID, record.Number, record.State, err))
	}
}

func (w *outputWriter) write(record attemptArtifact) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Encoding keeps arbitrary execution IDs inside one path component. Test
	// names and Case IDs never become paths. RunID remains unchanged in JSON.
	key := hex.EncodeToString([]byte(record.RunID))
	dir := filepath.Join(w.dir, "processes", record.ProcessID, "runs", key)
	attemptPath := filepath.Join(dir, fmt.Sprintf("attempt-%d.json", record.Number))
	if err := writeArtifact(attemptPath, record); err != nil {
		return err
	}

	// Attempts are authoritative. Rebuilding from their published snapshots
	// also supports retries recorded by different Plugin instances.
	attempts, err := readRunAttempts(dir, record.RunID, record.ProcessID)
	if err != nil {
		return err
	}
	run := newRunArtifact(record.RunID, attempts)
	return writeArtifact(filepath.Join(dir, "run.json"), run)
}
