package teststats

import "fmt"

// LoadResult contains the usable data and diagnostics from an output directory.
// Stats includes finished and finalized attempts. Running attempts have no
// outcome and appear only in Unfinished. Read files after test processes exit.
type LoadResult struct {
	Stats      *Stats
	Unfinished []UnfinishedAttempt
	Issues     []LoadIssue
}

// UnfinishedAttempt is a valid snapshot whose status is not finalized.
// A finished attempt also contributes its provisional outcome to Stats.
type UnfinishedAttempt struct {
	Path    string
	State   AttemptState
	Attempt Attempt
}

// LoadIssue describes a skipped file, unreadable directory, or missing attempt.
// Path is relative to the directory passed to Load and uses slash separators.
type LoadIssue struct {
	Path string
	Err  error
}

func (i LoadIssue) Error() string {
	return fmt.Sprintf("%s: %v", i.Path, i.Err)
}

func (i LoadIssue) Unwrap() error {
	return i.Err
}

// Incomplete reports no usable attempts, unfinalized attempts, or load issues.
// A false result describes only the observed files. It does not prove that all
// expected tests ran or that the test process exited successfully.
func (r LoadResult) Incomplete() bool {
	if r.Stats == nil || len(r.Unfinished) > 0 || len(r.Issues) > 0 {
		return true
	}
	return len(r.Stats.Attempts()) == 0
}
