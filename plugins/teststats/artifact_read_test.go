package teststats

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadRunAttempts_MissingDirectory(t *testing.T) {
	attempts, err := readRunAttempts(filepath.Join(t.TempDir(), "missing"), "run", "process")
	require.ErrorIs(t, err, fs.ErrNotExist)
	assert.Nil(t, attempts)
}

func TestReadRunAttempts_UnreadableAttempt(t *testing.T) {
	dir := t.TempDir()
	// A directory with the expected filename can be listed but cannot be read as a file.
	require.NoError(t, os.Mkdir(filepath.Join(dir, "attempt-1.json"), 0755))
	attempts, err := readRunAttempts(dir, "run", "process")
	require.Error(t, err)
	assert.Nil(t, attempts, "an unreadable attempt must not disappear from a successful summary")
}
