package testjunit

import (
	"bytes"
	"encoding/xml"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stagedFile struct {
	data               bytes.Buffer
	writeErr, closeErr error
	closes             int
}

func (f *stagedFile) Name() string   { return filepath.Join("reports", "temporary.xml") }
func (f *stagedFile) String() string { return f.data.String() }
func (f *stagedFile) Write(p []byte) (int, error) {
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	return f.data.Write(p)
}
func (f *stagedFile) Close() error { f.closes++; return f.closeErr }

func TestFileReplacementFailures(t *testing.T) {
	diskFull := errors.New("disk full")
	closeFailure := errors.New("close failed")
	denied := errors.New("permission denied")
	renameFailure := errors.New("rename failed")
	removeFailure := errors.New("remove failed")
	for _, tc := range []struct {
		name                                                          string
		mkdirErr, createErr, writeErr, closeErr, renameErr, removeErr error
		wantErrors                                                    []error
	}{
		{name: "directory", mkdirErr: denied, wantErrors: []error{denied}},
		{name: "create", createErr: denied, wantErrors: []error{denied}},
		{name: "write", writeErr: diskFull, wantErrors: []error{diskFull}},
		{name: "write close and cleanup", writeErr: diskFull, closeErr: closeFailure, removeErr: removeFailure, wantErrors: []error{diskFull, closeFailure, removeFailure}},
		{name: "close", closeErr: closeFailure, wantErrors: []error{closeFailure}},
		{name: "rename", renameErr: renameFailure, wantErrors: []error{renameFailure}},
		{name: "rename and cleanup", renameErr: renameFailure, removeErr: removeFailure, wantErrors: []error{renameFailure, removeFailure}},
		{name: "rename removed temporary path", removeErr: os.ErrNotExist},
		{name: "cleanup after rename", removeErr: removeFailure, wantErrors: []error{removeFailure}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			staged := &stagedFile{writeErr: tc.writeErr, closeErr: tc.closeErr}
			destination := "previous report"
			creates, removals, renames := 0, 0, 0
			ops := fileOperations{
				mkdirAll: func(string, fs.FileMode) error { return tc.mkdirErr },
				createTemp: func(dir, _ string) (reportFile, error) {
					creates++
					assert.Equal(t, "reports", dir, "temporary file in different directory")
					return staged, tc.createErr
				},
				remove: func(path string) error {
					removals++
					assert.Equal(t, staged.Name(), path, "removed wrong path")
					return tc.removeErr
				},
				rename: func(old, new string) error {
					renames++
					assert.Equal(t, 1, staged.closes, "file must be closed before rename")
					assert.Equal(t, staged.Name(), old)
					assert.Equal(t, filepath.Join("reports", "junit.xml"), new)
					if tc.renameErr == nil {
						destination = staged.String()
					}
					return tc.renameErr
				},
			}
			err := NewReporter().writeFile(filepath.Join("reports", "junit.xml"), ops)
			if len(tc.wantErrors) == 0 {
				require.NoError(t, err)
			}
			for _, want := range tc.wantErrors {
				require.ErrorIs(t, err, want)
			}
			created := tc.mkdirErr == nil && tc.createErr == nil
			if !created {
				assert.Zero(t, staged.closes, "closed a file that was not created")
				assert.Zero(t, removals, "removed a file that was not created")
				assert.Zero(t, renames, "renamed a file that was not created")
			}
			if tc.mkdirErr != nil {
				assert.Zero(t, creates, "created file after directory failure")
			}
			if created {
				assert.Equal(t, 1, staged.closes)
				assert.Equal(t, 1, removals)
			}
			if tc.writeErr != nil || tc.closeErr != nil {
				assert.Zero(t, renames, "replaced report before a successful write and close")
			}
			replaced := created && tc.writeErr == nil && tc.closeErr == nil && tc.renameErr == nil
			if !replaced {
				assert.Equal(t, "previous report", destination, "previous report damaged")
			}
			if replaced {
				var doc xmlSuites
				require.NoError(t, xml.Unmarshal([]byte(destination), &doc))
			}
		})
	}
}

func TestWriteFileRenameFailureCleansTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	require.Error(t, NewReporter().WriteFile(dir), "replaced a directory with XML")
	files, err := filepath.Glob(filepath.Join(filepath.Dir(dir), ".testjunit-*.xml"))
	require.NoError(t, err)
	assert.Empty(t, files, "temporary files leaked")
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir(), "destination directory damaged")
}
