package teststats

import (
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

func writeArtifact(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return replaceFile(path, append(data, '\n'), fileOperations{
		mkdirAll: os.MkdirAll,
		createTemp: func(dir, pattern string) (artifactFile, error) {
			return os.CreateTemp(dir, pattern)
		},
		rename: os.Rename,
		remove: os.Remove,
	})
}

type artifactFile interface {
	io.WriteCloser
	Name() string
	Sync() error
}

type fileOperations struct {
	mkdirAll   func(string, fs.FileMode) error
	createTemp func(string, string) (artifactFile, error)
	rename     func(string, string) error
	remove     func(string) error
}

// replaceFile publishes a whole snapshot only after a successful write, sync,
// and close. Atomic replacement follows the platform's os.Rename guarantees.
func replaceFile(path string, data []byte, files fileOperations) (err error) {
	dir := filepath.Dir(path)
	if err := files.mkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := files.createTemp(dir, ".teststats-*.tmp")
	if err != nil {
		return err
	}
	defer func() {
		if removeErr := files.remove(f.Name()); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}()
	written, err := f.Write(data)
	if err != nil {
		return errors.Join(err, f.Close())
	}
	if written != len(data) {
		return errors.Join(io.ErrShortWrite, f.Close())
	}
	if err := f.Sync(); err != nil {
		return errors.Join(err, f.Close())
	}
	if err := f.Close(); err != nil {
		return err
	}
	return files.rename(f.Name(), path)
}
