package testjunit

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecorderReconcilesAtParentCleanup(t *testing.T) {
	reporter := NewReporter()
	late := &fixedOutcome{}
	t.Run("parent", func(t *testing.T) {
		r := &recorder{
			cfg: &axiom.Config{RootT: t}, reporter: reporter,
			attempt: &attempt{TestName: "child", Start: time.Now()},
		}
		r.finish(late)
		initial := reporter.snapshot()
		require.Len(t, initial, 1)
		assert.Equal(t, statusPassed, initial[0].Status)
		late.failed = true
		beforeCleanup := reporter.snapshot()
		require.Len(t, beforeCleanup, 1)
		assert.Equal(t, statusPassed, beforeCleanup[0].Status, "snapshot read testing.T before parent cleanup")
	})
	final := reporter.snapshot()
	require.Len(t, final, 1)
	assert.Equal(t, statusFailed, final[0].Status, "late failure was not reconciled")
}
