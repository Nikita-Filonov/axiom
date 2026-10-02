package testjunit

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteFile(t *testing.T) {
	r := NewReporter()
	path := filepath.Join(t.TempDir(), "nested", "junit.xml")
	require.NoError(t, r.WriteFile(path))
	first, err := os.ReadFile(path)
	require.NoError(t, err)
	require.NoError(t, r.WriteFile(path))
	second, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, first, second, "replacement changed report")
	matches, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".testjunit-*.xml"))
	require.NoError(t, err)
	assert.Empty(t, matches, "temporary files remain")
}

func TestWriteFileErrors(t *testing.T) {
	r := NewReporter()
	require.Error(t, r.WriteFile(""), "empty path accepted")
	var nilReporter *Reporter
	require.Error(t, nilReporter.WriteFile("report.xml"), "nil reporter accepted")
	parent := filepath.Join(t.TempDir(), "parent-file")
	require.NoError(t, os.WriteFile(parent, []byte("unchanged"), 0600))
	require.Error(t, r.WriteFile(filepath.Join(parent, "report.xml")), "non-directory parent accepted")
	contents, err := os.ReadFile(parent)
	require.NoError(t, err)
	assert.Equal(t, "unchanged", string(contents), "parent file changed")
}
