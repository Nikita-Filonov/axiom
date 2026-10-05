package testtimeout

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_ContextDeadlineMatchesAttemptAndCancelsOnCompletion(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(Plugin(WithTimeout(time.Second), WithContextDeadline())),
	)

	var contexts []context.Context
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("deadline")), func(cfg *axiom.Config) {
		contexts = []context.Context{cfg.Context.Raw, cfg.Context.DB, cfg.Context.MQ, cfg.Context.RPC}
		firstDeadline, ok := contexts[0].Deadline()
		require.True(t, ok)
		for _, ctx := range contexts {
			deadline, ok := ctx.Deadline()
			require.True(t, ok)
			assert.Equal(t, firstDeadline, deadline)
			assert.NoError(t, ctx.Err())
		}
	})

	require.Len(t, contexts, 4)
	for _, ctx := range contexts {
		assert.ErrorIs(t, ctx.Err(), context.Canceled)
	}
}

func TestPlugin_ContextDeadlinePreservesEarlierParent(t *testing.T) {
	parentDeadline := time.Now().Add(time.Minute)
	parent, cancel := context.WithDeadline(context.Background(), parentDeadline)
	defer cancel()

	runner := axiom.NewRunner(
		axiom.WithRunnerContext(axiom.WithContextDB(parent)),
		axiom.WithRunnerPlugins(Plugin(WithTimeout(2*time.Minute), WithContextDeadline())),
	)
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("earlier parent")), func(cfg *axiom.Config) {
		deadline, ok := cfg.Context.DB.Deadline()
		require.True(t, ok)
		assert.Equal(t, parentDeadline, deadline)
	})

	assert.NoError(t, parent.Err(), "the plugin must not cancel the supplied parent")
}

func TestPlugin_WithoutContextDeadlineLeavesContextsUnchanged(t *testing.T) {
	parent := context.WithValue(context.Background(), contextKey("source"), "runner")
	runner := axiom.NewRunner(
		axiom.WithRunnerContext(axiom.WithContextRaw(parent)),
		axiom.WithRunnerPlugins(Plugin(WithTimeout(time.Second))),
	)

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("opt in")), func(cfg *axiom.Config) {
		for _, ctx := range []context.Context{cfg.Context.Raw, cfg.Context.DB, cfg.Context.MQ, cfg.Context.RPC} {
			assert.Same(t, parent, ctx)
			_, hasDeadline := ctx.Deadline()
			assert.False(t, hasDeadline)
		}
	})

	assert.NoError(t, parent.Err())
}

func TestPlugin_SeparateCasesGetFreshContexts(t *testing.T) {
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(Plugin(WithTimeout(time.Second), WithContextDeadline())),
	)

	var first, second context.Context
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("first")), func(cfg *axiom.Config) {
		first = cfg.Context.RPC
		assert.NoError(t, first.Err())
	})
	require.NotNil(t, first)
	assert.ErrorIs(t, first.Err(), context.Canceled)

	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("second")), func(cfg *axiom.Config) {
		second = cfg.Context.RPC
		assert.NoError(t, second.Err(), "the second case must not inherit the first case's cancellation")
	})
	require.NotNil(t, second)
	assert.NotSame(t, first, second)
	assert.ErrorIs(t, second.Err(), context.Canceled)
}

func TestApplyContextDeadline_PreservesIndependentParents(t *testing.T) {
	key := contextKey("source")
	parents := []context.Context{
		context.WithValue(context.Background(), key, "raw"),
		context.WithValue(context.Background(), key, "db"),
		context.WithValue(context.Background(), key, "mq"),
		context.WithValue(context.Background(), key, "rpc"),
	}
	cfg := &axiom.Config{Context: axiom.Context{
		Raw: parents[0], DB: parents[1], MQ: parents[2], RPC: parents[3],
	}}

	cancel := applyContextDeadline(cfg, time.Now().Add(time.Minute))
	derived := []context.Context{cfg.Context.Raw, cfg.Context.DB, cfg.Context.MQ, cfg.Context.RPC}
	for i, ctx := range derived {
		assert.Equal(t, parents[i].Value(key), ctx.Value(key))
		assert.NotSame(t, parents[i], ctx)
	}
	cancel()

	for i, ctx := range derived {
		assert.ErrorIs(t, ctx.Err(), context.Canceled)
		assert.NoError(t, parents[i].Err(), "canceling a derived context must not cancel its parent")
	}
}

func TestPlugin_ContextDeadlineStartsAfterParallelPause(t *testing.T) {
	const budget = time.Second
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(Plugin(WithTimeout(budget), WithContextDeadline())),
	)
	var parentExit atomic.Pointer[time.Time]

	t.Run("parent", func(parent *testing.T) {
		parallelCase := axiom.NewCase(
			axiom.WithCaseName("parallel"),
			axiom.WithCaseParallel(axiom.WithParallelEnabled()),
		)
		runner.RunCase(parent, parallelCase, func(cfg *axiom.Config) {
			resumedAfter := parentExit.Load()
			if resumedAfter == nil {
				cfg.T().Error("parallel case ran before its parent returned")
				return
			}
			deadline, ok := cfg.Context.RPC.Deadline()
			if !ok {
				cfg.T().Error("RPC context has no deadline")
				return
			}
			if deadline.Before(resumedAfter.Add(budget)) {
				cfg.T().Errorf("deadline %s started before the parallel pause ended at %s", deadline, *resumedAfter)
			}
		})

		// Parallel subtests resume after their parent returns. Their deadline
		// must be based on a time later than this marker.
		at := time.Now()
		parentExit.Store(&at)
	})
}

type contextKey string
