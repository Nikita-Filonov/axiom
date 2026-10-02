package testjunit

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// WriteFile writes a JUnit XML snapshot through a temporary file in the same
// directory, then renames it to path. Parent directories are created as needed.
// Call it after tests finish. Replacement of an existing path depends on the
// platform's os.Rename behavior.
func (r *Reporter) WriteFile(path string) error {
	return r.writeFile(path, fileOperations{
		mkdirAll:   os.MkdirAll,
		createTemp: func(dir, pattern string) (reportFile, error) { return os.CreateTemp(dir, pattern) },
		remove:     os.Remove,
		rename:     os.Rename,
	})
}

// fileOperations is the I/O boundary for staged report replacement.
// Passing it per call keeps failure tests independent and safe to run in parallel.
type fileOperations struct {
	mkdirAll   func(string, fs.FileMode) error
	createTemp func(string, string) (reportFile, error)
	remove     func(string) error
	rename     func(string, string) error
}

type reportFile interface {
	io.WriteCloser
	Name() string
}

func (r *Reporter) writeFile(path string, files fileOperations) (err error) {
	if r == nil {
		return errors.New("testjunit: nil reporter")
	}
	if path == "" {
		return errors.New("testjunit: empty report path")
	}
	dir := filepath.Dir(path)
	if err := files.mkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := files.createTemp(dir, ".testjunit-*.xml")
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := files.remove(f.Name()); cleanupErr != nil && !errors.Is(cleanupErr, os.ErrNotExist) {
			err = errors.Join(err, cleanupErr)
		}
	}()
	if err := r.Write(f); err != nil {
		return errors.Join(err, f.Close())
	}
	if err := f.Close(); err != nil {
		return err
	}
	return files.rename(f.Name(), path)
}
