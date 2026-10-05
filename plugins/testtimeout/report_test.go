package testtimeout

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReportTimeout_DumpsGoroutineArtefact(t *testing.T) {
	var captured []axiom.Artefact
	c := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeArtefactSink(func(a axiom.Artefact) { captured = append(captured, a) }),
		),
	}

	reportTimeout(c, Config{Timeout: time.Second, DumpGoroutines: true})

	require.Len(t, captured, 1)
	assert.Equal(t, "testtimeout-goroutines.txt", captured[0].Name)
}

func TestReportTimeout_FailsThroughT(t *testing.T) {
	orphan := &testing.T{}

	reportTimeout(&axiom.Config{RootT: orphan}, Config{Timeout: time.Second})

	assert.True(t, orphan.Failed(), "the case must be marked failed on timeout")
}

func TestGoroutineDump_ContainsGoroutines(t *testing.T) {
	assert.Contains(t, goroutineDump(), "goroutine")
}
