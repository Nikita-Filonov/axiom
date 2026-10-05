package testtimeout

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestRunWithTimeout_WithoutContextDeadlineLeavesParentRunning(t *testing.T) {
	// c.T() == nil, so the timeout branch is exercised without failing anything.
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &axiom.Config{Context: axiom.Context{Raw: parent}}

	release := make(chan struct{})
	finished := make(chan struct{})

	runWithTimeout(c, Config{Timeout: 5 * time.Millisecond, Message: "custom deadline"}, func(*axiom.Config) {
		<-release
		close(finished)
	})

	assert.Same(t, parent, c.Context.Raw)
	assert.NoError(t, parent.Err(), "the default timeout must not cancel supplied contexts")
	close(release)
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Error("the body did not finish after release")
	}
}

func TestRunWithTimeout_TimeoutCancelsContextsWhileBodyContinues(t *testing.T) {
	orphan := &testing.T{}
	cfg := &axiom.Config{RootT: orphan}
	started := make(chan struct{})
	release := make(chan struct{})
	finished := make(chan struct{})
	returned := make(chan struct{})
	var once sync.Once
	releaseBody := func() { once.Do(func() { close(release) }) }
	defer releaseBody()

	go func() {
		runWithTimeout(cfg, Config{Timeout: 50 * time.Millisecond, ContextDeadline: true}, func(*axiom.Config) {
			close(started)
			<-release // Deliberately ignore context cancellation.
			close(finished)
		})
		close(returned)
	}()

	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("the body did not start")
	}
	select {
	case <-returned:
	case <-time.After(time.Second):
		t.Fatal("the timeout wrapper did not return")
	}

	assert.True(t, orphan.Failed())
	// The wrapper's timer may cancel the context before its own deadline timer
	// fires. Both Canceled and DeadlineExceeded mean the context was stopped.
	for _, ctx := range []context.Context{cfg.Context.Raw, cfg.Context.DB, cfg.Context.MQ, cfg.Context.RPC} {
		assert.Error(t, ctx.Err())
	}
	select {
	case <-finished:
		t.Error("the body should still be waiting after the wrapper returns")
	default:
	}

	releaseBody()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Error("the body did not finish after release")
	}
}

func TestRunWithTimeout_PanicCancelsDerivedContexts(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := &axiom.Config{Context: axiom.Context{Raw: parent}}
	var child context.Context

	assert.PanicsWithValue(t, "boom", func() {
		runWithTimeout(cfg, Config{Timeout: time.Second, ContextDeadline: true}, func(cfg *axiom.Config) {
			child = cfg.Context.RPC
			panic("boom")
		})
	})

	require.NotNil(t, child)
	assert.ErrorIs(t, child.Err(), context.Canceled)
	assert.NoError(t, parent.Err())
}

func TestRunWithTimeout_EarlierParentCancellationDoesNotFailAttempt(t *testing.T) {
	parent, cancel := context.WithCancel(context.Background())
	cancel()
	orphan := &testing.T{}
	cfg := &axiom.Config{RootT: orphan, Context: axiom.Context{RPC: parent}}

	runWithTimeout(cfg, Config{Timeout: time.Second, ContextDeadline: true}, func(cfg *axiom.Config) {
		<-cfg.Context.RPC.Done()
	})

	assert.False(t, orphan.Failed(), "an existing parent cancellation is not the test timeout")
	assert.ErrorIs(t, cfg.Context.RPC.Err(), context.Canceled)
}
