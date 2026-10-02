package testotel_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testotel"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

func recordedProvider(t *testing.T) (*sdktrace.TracerProvider, *tracetest.SpanRecorder) {
	t.Helper()
	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	t.Cleanup(func() { require.NoError(t, provider.Shutdown(context.Background())) })
	return provider, recorder
}

func attributes(span sdktrace.ReadOnlySpan) map[string]attribute.Value {
	values := make(map[string]attribute.Value)
	for _, item := range span.Attributes() {
		values[string(item.Key)] = item.Value
	}
	return values
}

func TestAttemptSpanAndContext(t *testing.T) {
	provider, recorder := recordedProvider(t)
	parentContext, parent := provider.Tracer("test").Start(t.Context(), "parent")
	runner := axiom.NewRunner(
		axiom.WithRunnerContext(axiom.WithContextRaw(parentContext)),
		axiom.WithRunnerPlugins(testotel.Plugin(testotel.WithTracerProvider(provider))),
	)
	var cleanupFinished time.Time
	var attemptContext context.Context
	caseDef := axiom.NewCase(axiom.WithCaseID("case-1"), axiom.WithCaseName("works"))
	runner.RunCase(t, caseDef, func(cfg *axiom.Config) {
		attemptContext = testotel.Context(cfg)
		require.True(t, trace.SpanFromContext(attemptContext).SpanContext().IsValid())
		require.Equal(t, parent.SpanContext().TraceID(), trace.SpanFromContext(attemptContext).SpanContext().TraceID())
		require.NotEqual(t, cfg.Context.Raw, attemptContext)
		cfg.Step("prepare", func() {
			_, child := provider.Tracer("application").Start(testotel.Context(cfg), "client request")
			child.End()
		})
		cfg.Log(axiom.NewInfoLog("secret log body"))
		cfg.Event(axiom.NewEvent(axiom.EventTypeAssert, axiom.WithEventMessage("secret assertion")))
		cfg.T().Cleanup(func() { cleanupFinished = time.Now() })
	})
	parent.End()

	spans := recorder.Ended()
	require.Len(t, spans, 3)
	var attempt, child sdktrace.ReadOnlySpan
	for _, span := range spans {
		switch span.Name() {
		case "axiom.test":
			attempt = span
		case "client request":
			child = span
		}
	}
	require.NotNil(t, attempt)
	require.NotNil(t, child)
	require.Equal(t, parent.SpanContext().SpanID(), attempt.Parent().SpanID())
	require.Equal(t, attempt.SpanContext().SpanID(), child.Parent().SpanID())
	require.False(t, attempt.EndTime().Before(cleanupFinished))
	require.Equal(t, codes.Unset, attempt.Status().Code)
	attrs := attributes(attempt)
	require.Equal(t, "passed", attrs["axiom.test.status"].AsString())
	require.Equal(t, "works", attrs["axiom.case.name"].AsString())
	require.Equal(t, "case-1", attrs["axiom.case.id"].AsString())
	require.Equal(t, int64(1), attrs["axiom.execution.attempt"].AsInt64())
	require.NotEmpty(t, attrs["axiom.execution.id"].AsString())
	require.Equal(t, "github.com/Nikita-Filonov/axiom/plugins/testotel", attempt.InstrumentationScope().Name)
	require.Equal(t, []string{"step.start", "step.finish"}, eventNames(attempt.Events()))
	encoded := fmt.Sprint(attempt.Attributes(), attempt.Events(), attempt.Status())
	require.NotContains(t, encoded, "secret log body")
	require.NotContains(t, encoded, "secret assertion")
}

func eventNames(events []sdktrace.Event) []string {
	names := make([]string, 0, len(events))
	for _, event := range events {
		names = append(names, event.Name)
	}
	return names
}

func TestPolicySkipSpan(t *testing.T) {
	provider, recorder := recordedProvider(t)
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(testotel.Plugin(testotel.WithTracerProvider(provider))))
	runner.RunCase(t, axiom.NewCase(
		axiom.WithCaseName("skipped"),
		axiom.WithCaseSkip(axiom.SkipBecause("secret reason")),
	), func(*axiom.Config) { t.Fatal("skipped test body must not run") })

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	attrs := attributes(spans[0])
	require.Equal(t, "skipped", attrs["axiom.test.status"].AsString())
	require.Equal(t, int64(1), attrs["axiom.execution.attempt"].AsInt64())
	require.Equal(t, []string{"case.skip"}, eventNames(spans[0].Events()))
	require.NotContains(t, spans[0].Status().Description, "secret reason")
}

func TestBodySkipSpan(t *testing.T) {
	provider, recorder := recordedProvider(t)
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(testotel.Plugin(testotel.WithTracerProvider(provider))))
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("body skip")), func(cfg *axiom.Config) {
		cfg.T().Skip("secret runtime reason")
	})
	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "skipped", attributes(spans[0])["axiom.test.status"].AsString())
	require.NotContains(t, fmt.Sprint(spans[0].Attributes(), spans[0].Events()), "secret runtime reason")
}

func TestParallelRetryPolicySkipUsesPlanningConfig(t *testing.T) {
	provider, recorder := recordedProvider(t)
	runner := axiom.NewRunner(axiom.WithRunnerPlugins(testotel.Plugin(testotel.WithTracerProvider(provider))))
	t.Run("group", func(group *testing.T) {
		runner.RunCase(group, axiom.NewCase(
			axiom.WithCaseName("parallel skip"),
			axiom.WithCaseParallel(axiom.WithParallelEnabled()),
			axiom.WithCaseRetry(axiom.WithRetryTimes(2)),
			axiom.WithCaseSkip(axiom.SkipBecause("policy")),
		), func(*axiom.Config) { group.Fatal("skipped test body must not run") })
	})
	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, "skipped", attributes(spans[0])["axiom.test.status"].AsString())
	require.Equal(t, int64(1), attributes(spans[0])["axiom.execution.attempt"].AsInt64())
}

func TestLifecycleFailureAndDuplicateInstallation(t *testing.T) {
	provider, recorder := recordedProvider(t)
	plugin := testotel.Plugin(testotel.WithTracerProvider(provider))
	t.Run("attempt", func(sub *testing.T) {
		cfg := &axiom.Config{RootT: sub, Case: &axiom.Case{Name: "manual"}}
		plugin(cfg)
		plugin(cfg)
		cfg.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
		cfg.Event(axiom.NewEvent(axiom.EventTypeFixtureSetupFailed,
			axiom.WithEventName("database"), axiom.WithEventMessage("secret password")))
		cfg.Event(axiom.NewEvent(axiom.EventTypeCaseFinish))
	})

	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Equal(t, codes.Error, spans[0].Status().Code)
	require.Equal(t, "failed", attributes(spans[0])["axiom.test.status"].AsString())
	require.Equal(t, []string{"fixture.setup.failed"}, eventNames(spans[0].Events()))
	require.Equal(t, "database", spans[0].Events()[0].Attributes[0].Value.AsString())
	require.NotContains(t, spans[0].Status().Description, "secret password")
	require.NotContains(t, fmt.Sprint(spans[0].Attributes(), spans[0].Events()), "secret password")
}

func TestContextFallbackAndValidation(t *testing.T) {
	parent, cancel := context.WithCancel(t.Context())
	defer cancel()
	cfg := &axiom.Config{Context: axiom.Context{Raw: parent}}
	require.Same(t, parent, testotel.Context(cfg))
	require.NotNil(t, testotel.Context(&axiom.Config{}))
	require.PanicsWithValue(t, "testotel: nil config", func() { testotel.Context(nil) })
	require.PanicsWithValue(t, "testotel: nil option", func() { testotel.Plugin(nil) })
	require.PanicsWithValue(t, "testotel: nil tracer provider", func() {
		testotel.Plugin(testotel.WithTracerProvider(nil))
	})
	require.PanicsWithValue(t, "testotel: nil config", func() { testotel.Plugin()(nil) })
}

func TestEventsOutsideAttemptDoNotCreateOrChangeSpans(t *testing.T) {
	provider, recorder := recordedProvider(t)
	plugin := testotel.Plugin(testotel.WithTracerProvider(provider))
	noTest := &axiom.Config{}
	plugin(noTest)
	noTest.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
	require.Empty(t, recorder.Started())

	var cfg *axiom.Config
	t.Run("attempt", func(sub *testing.T) {
		cfg = &axiom.Config{RootT: sub}
		plugin(cfg)
		cfg.Event(axiom.NewEvent(axiom.EventTypeStepStart, axiom.WithEventName("too early")))
		cfg.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
		cfg.Event(axiom.NewEvent(axiom.EventTypeCaseStart))
	})
	spans := recorder.Ended()
	require.Len(t, spans, 1)
	require.Empty(t, spans[0].Events())
	cfg.Event(axiom.NewEvent(axiom.EventTypeStepFinish, axiom.WithEventName("too late")))
	require.Empty(t, spans[0].Events())
}

type probeSpan struct {
	RunID   string `json:"run_id"`
	Attempt int    `json:"attempt"`
	Status  string `json:"status"`
}

func TestRetryFailureSpans(t *testing.T) {
	for _, mode := range []string{"body-error", "cleanup-error"} {
		t.Run(mode, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "spans.json")
			exe, err := os.Executable()
			require.NoError(t, err)
			cmd := exec.Command(exe, "-test.run=^TestFailureProbe$")
			cmd.Env = append(os.Environ(), "AXIOM_OTEL_PROBE="+path, "AXIOM_OTEL_PROBE_MODE="+mode)
			output, err := cmd.CombinedOutput()
			require.Error(t, err, "the first Go test failure must remain a failure: %s", output)

			data, err := os.ReadFile(path)
			require.NoError(t, err, "%s", output)
			var spans []probeSpan
			require.NoError(t, json.Unmarshal(data, &spans))
			require.Len(t, spans, 2, "%s", output)
			slices.SortFunc(spans, func(a, b probeSpan) int { return a.Attempt - b.Attempt })
			require.NotEmpty(t, spans[0].RunID)
			require.Equal(t, spans[0].RunID, spans[1].RunID)
			require.Equal(t, []probeSpan{
				{RunID: spans[0].RunID, Attempt: 1, Status: "failed"},
				{RunID: spans[0].RunID, Attempt: 2, Status: "passed"},
			}, spans)
		})
	}
}

func TestFailureProbe(t *testing.T) {
	path := os.Getenv("AXIOM_OTEL_PROBE")
	if path == "" {
		t.Skip("subprocess only")
	}
	provider, recorder := recordedProvider(t)
	t.Cleanup(func() {
		var spans []probeSpan
		for _, span := range recorder.Ended() {
			attrs := attributes(span)
			spans = append(spans, probeSpan{
				RunID:   attrs["axiom.execution.id"].AsString(),
				Attempt: int(attrs["axiom.execution.attempt"].AsInt64()),
				Status:  attrs["axiom.test.status"].AsString(),
			})
		}
		data, err := json.Marshal(spans)
		if err != nil {
			t.Error(err)
			return
		}
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Error(err)
		}
	})
	var attempts int
	runner := axiom.NewRunner(
		axiom.WithRunnerRetry(axiom.WithRetryTimes(2)),
		axiom.WithRunnerPlugins(testotel.Plugin(testotel.WithTracerProvider(provider))),
	)
	runner.RunCase(t, axiom.NewCase(axiom.WithCaseName("retry")), func(cfg *axiom.Config) {
		attempts++
		if attempts == 1 {
			if os.Getenv("AXIOM_OTEL_PROBE_MODE") == "cleanup-error" {
				cfg.T().Cleanup(func() { cfg.T().Error("intentional cleanup failure") })
			} else {
				cfg.T().Error("intentional failure")
			}
		}
	})
}
