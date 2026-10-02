package testotel_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testotel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_ReinstallationPreservesProviderAndContext(t *testing.T) {
	firstProvider, firstRecorder := recordedProvider(t)
	secondProvider, secondRecorder := recordedProvider(t)
	firstPlugin := testotel.Plugin(testotel.WithTracerProvider(firstProvider))
	secondPlugin := testotel.Plugin(testotel.WithTracerProvider(secondProvider))
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(firstPlugin))
	caseDef := axiom.NewCase(
		axiom.WithCaseName("traced"),
		axiom.WithCasePlugins(secondPlugin),
	)

	var finished *axiom.Config
	for range 2 {
		runner.RunCase(t, caseDef, func(cfg *axiom.Config) {
			finished = cfg
			before := testotel.Context(cfg)
			firstPlugin(cfg)
			secondPlugin(cfg)
			assert.Len(cfg.T(), cfg.Runtime.EventSinks, 1)
			assert.Same(cfg.T(), before, testotel.Context(cfg))
			cfg.Step("work", func() {})
		})
	}

	spans := firstRecorder.Ended()
	require.Len(t, spans, 2)
	assert.NotEqual(t, spans[0].SpanContext().SpanID(), spans[1].SpanContext().SpanID())
	for _, span := range spans {
		assert.Equal(t, []string{"step.start", "step.finish"}, eventNames(span.Events()))
	}
	assert.Empty(t, secondRecorder.Started(), "the first installation's provider must win")

	// Reinstallation after cleanup must not create a fresh attempt or span.
	secondPlugin(finished)
	finished.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
	assert.Len(t, firstRecorder.Started(), 2)
	assert.Empty(t, secondRecorder.Started())
}
