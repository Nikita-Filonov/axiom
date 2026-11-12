package teststats

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errFileFailure = errors.New("injected file failure")

type failingFile struct {
	*os.File
	stage string
	calls *[]string
}

func (f *failingFile) Write(data []byte) (int, error) {
	*f.calls = append(*f.calls, "write")
	if f.stage == "write" {
		n, _ := f.File.Write(data[:len(data)/2])
		return n, errFileFailure
	}
	if f.stage == "short write" {
		return f.File.Write(data[:len(data)/2])
	}
	return f.File.Write(data)
}

func (f *failingFile) Sync() error {
	*f.calls = append(*f.calls, "sync")
	if f.stage == "sync" {
		return errFileFailure
	}
	return f.File.Sync()
}

func (f *failingFile) Close() error {
	*f.calls = append(*f.calls, "close")
	err := f.File.Close()
	if f.stage == "close" {
		return errors.Join(err, errFileFailure)
	}
	return err
}

func TestReplaceFile_PreservesPublishedSnapshotOnFailure(t *testing.T) {
	for _, stage := range []string{"mkdir", "create", "write", "short write", "sync", "close", "rename", "success"} {
		t.Run(stage, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "attempt-1.json")
			require.NoError(t, os.WriteFile(path, []byte("previous"), 0600))
			var calls []string
			err := replaceFile(path, []byte("replacement"), fileOperations{
				mkdirAll: func(path string, mode fs.FileMode) error {
					if stage == "mkdir" {
						return errFileFailure
					}
					return os.MkdirAll(path, mode)
				},
				createTemp: func(dir, pattern string) (artifactFile, error) {
					if stage == "create" {
						return nil, errFileFailure
					}
					f, err := os.CreateTemp(dir, pattern)
					if err != nil {
						return nil, err
					}
					return &failingFile{
						File:  f,
						stage: stage,
						calls: &calls,
					}, nil
				},
				rename: func(old, new string) error {
					calls = append(calls, "rename")
					if stage == "rename" {
						return errFileFailure
					}
					return os.Rename(old, new)
				},
				remove: os.Remove,
			})
			data, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			if stage == "success" {
				require.NoError(t, err)
				assert.Equal(t, "replacement", string(data))
				assert.Equal(t, []string{"write", "sync", "close", "rename"}, calls)
			} else {
				if stage == "short write" {
					assert.ErrorIs(t, err, io.ErrShortWrite)
				} else {
					assert.ErrorIs(t, err, errFileFailure)
				}
				assert.Equal(t, "previous", string(data))
			}
			entries, err := os.ReadDir(dir)
			require.NoError(t, err)
			require.Len(t, entries, 1, "temporary file was not removed")
			assert.Equal(t, "attempt-1.json", entries[0].Name())
		})
	}
}
