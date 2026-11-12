package teststats

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"
)

// Load restores statistics from an output directory written by WithOutputDir.
// It reads attempt files and ignores derived run.json snapshots. A missing or
// unreadable root returns an error. Errors below individual process directories
// are collected in Issues, so usable results from other files are retained.
// Load never writes files and does not change the test process exit code.
func Load(dir string) (*LoadResult, error) {
	if strings.TrimSpace(dir) == "" {
		return nil, errors.New("teststats: empty output directory")
	}
	result, err := loadFiles(os.DirFS(dir))
	if err != nil {
		return nil, fmt.Errorf("teststats: load %q: %w", dir, err)
	}
	return result, nil
}

type resultLoader struct {
	files  fs.FS
	result *LoadResult
}

func loadFiles(files fs.FS) (*LoadResult, error) {
	// Reading the root distinguishes a missing/unreadable path from an existing
	// directory with no exported attempts yet.
	if _, err := fs.ReadDir(files, "."); err != nil {
		return nil, err
	}
	processes, err := readArtifactDirectory(files, "processes")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	loader := resultLoader{
		files: files,
		result: &LoadResult{
			Stats: NewStats(),
		},
	}
	for _, process := range processes {
		processPath := path.Join("processes", process.Name())
		if loader.acceptDirectory(processPath, process) {
			loader.loadProcess(process.Name())
		}
	}
	return loader.result, nil
}

func (l *resultLoader) loadProcess(processID string) {
	runsPath := path.Join("processes", processID, "runs")
	runs, err := readArtifactDirectory(l.files, runsPath)
	if err != nil {
		l.issue(runsPath, err)
		return
	}
	runDirectories := 0
	for _, run := range runs {
		runPath := path.Join(runsPath, run.Name())
		if l.acceptDirectory(runPath, run) {
			runDirectories++
			l.loadRun(processID, run.Name())
		}
	}
	if runDirectories == 0 {
		l.issue(runsPath, errors.New("no run directories"))
	}
}

// The parent was checked during traversal. Inspecting its entry prevents
// ReadDir from following a symbolic link at the next directory level.
func readArtifactDirectory(files fs.FS, name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(files, path.Dir(name))
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.Name() != path.Base(name) {
			continue
		}
		if !entry.IsDir() {
			return nil, fmt.Errorf("%s: expected a directory, symbolic links are not supported", name)
		}
		return fs.ReadDir(files, name)
	}
	return nil, &fs.PathError{
		Op:   "readdir",
		Path: name,
		Err:  fs.ErrNotExist,
	}
}

func (l *resultLoader) acceptDirectory(path string, entry fs.DirEntry) bool {
	if entry.Type()&fs.ModeSymlink != 0 {
		l.issue(path, errors.New("symbolic links are not supported"))
		return false
	}
	return entry.IsDir()
}

func (l *resultLoader) issue(path string, err error) {
	l.result.Issues = append(l.result.Issues, LoadIssue{
		Path: path,
		Err:  err,
	})
}
