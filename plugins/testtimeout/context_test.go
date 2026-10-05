package testtimeout

import (
	"context"
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
