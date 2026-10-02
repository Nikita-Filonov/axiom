package testotel_test

import (
	"context"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testotel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestTracedCase(t *testing.T) {
	// This in-memory recorder stands in for an exporter.
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	// The test owns the provider and shuts it down after all subtests finish.
	t.Cleanup(func() {
		if err := provider.Shutdown(context.Background()); err != nil {
			t.Error(err)
		}
	})

	// Installing the plugin on the Runner traces every case attempt it runs.
	runner := axiom.NewRunner(
		axiom.WithRunnerPlugins(
			testotel.Plugin(testotel.WithTracerProvider(provider)),
		),
	)
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("request works")), func(cfg *axiom.Config) {
		// Pass the attempt context to instrumented clients. This child span
		// stands in for a span created by an HTTP or database client.
		ctx := testotel.Context(cfg)
		_, child := provider.Tracer("application").Start(ctx, "request")
		child.End()

		// Axiom step transitions appear as events on the attempt span.
		cfg.Step("check response", func() {})
	})

	// RunCase waits for the attempt's cleanup, so both spans have ended.
	if got := len(recorder.Ended()); got != 2 {
		t.Fatalf("got %d completed spans, want 2", got)
	}
}
