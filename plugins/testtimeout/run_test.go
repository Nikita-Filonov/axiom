package testtimeout

import (
	"context"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
)

func TestRunWithTimeout_ContextDeadlineFailsAttemptWhenBodyReturnsOnCancellation(t *testing.T) {
	orphan := &testing.T{}
	cfg := &axiom.Config{RootT: orphan}

	runWithTimeout(cfg, Config{Timeout: 20 * time.Millisecond, ContextDeadline: true}, func(cfg *axiom.Config) {
		<-cfg.Context.RPC.Done()
	})

	assert.True(t, orphan.Failed())
	assert.ErrorIs(t, cfg.Context.RPC.Err(), context.DeadlineExceeded)
}

func TestRunWithTimeout_RePanicsBodyPanic(t *testing.T) {
	c := &axiom.Config{}

	assert.PanicsWithValue(t, "boom", func() {
		runWithTimeout(c, Config{Timeout: time.Second}, func(*axiom.Config) {
			panic("boom")
		})
	})
}

func TestRunWithTimeout_TimesOut(t *testing.T) {
	// c.T() == nil, so the timeout branch is exercised without failing anything.
	c := &axiom.Config{}

	release := make(chan struct{})
	defer close(release)

	runWithTimeout(c, Config{Timeout: 5 * time.Millisecond, Message: "custom deadline"}, func(*axiom.Config) {
		<-release
	})
}
