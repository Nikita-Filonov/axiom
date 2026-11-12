package teststats

import (
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadResult_ZeroValueIsIncomplete(t *testing.T) {
	assert.True(t, (LoadResult{}).Incomplete())
}

func TestLoadIssue_PreservesUnderlyingError(t *testing.T) {
	issue := LoadIssue{
		Path: "attempt-1.json",
		Err:  fs.ErrPermission,
	}
	assert.EqualError(t, issue, "attempt-1.json: permission denied")
	assert.ErrorIs(t, issue, fs.ErrPermission)
	assert.NotErrorIs(t, issue, fs.ErrNotExist)
}
