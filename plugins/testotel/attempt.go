package testotel

import (
	"context"
	"sync"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

const (
	testSpanName              = "axiom.test"
	attributeTestName         = "axiom.test.name"
	attributeTestStatus       = "axiom.test.status"
	attributeExecutionAttempt = "axiom.execution.attempt"
	attributeExecutionID      = "axiom.execution.id"
	attributeCaseName         = "axiom.case.name"
	attributeCaseID           = "axiom.case.id"
	attributeEventName        = "axiom.event.name"

	statusPassed  = "passed"
	statusFailed  = "failed"
	statusSkipped = "skipped"
)

var attemptKey = axiom.NewLocalKey[*attempt]("testotel.attempt")

type attempt struct {
	cfg    *axiom.Config
	tracer trace.Tracer

	startOnce sync.Once
	mu        sync.Mutex
	ctx       context.Context
	span      trace.Span
	started   bool
	ended     bool
	failed    bool
	skipped   bool
}

func (a *attempt) context() context.Context {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.ctx
}

func (a *attempt) observe(event axiom.Event) {
	switch event.Type {
	case axiom.EventTypeCaseStart:
		a.start(false)
	case axiom.EventTypeCaseSkip:
		a.start(true)
	default:
		a.record(event)
	}
}

func (a *attempt) start(skipped bool) {
	t := a.cfg.T()
	if t == nil {
		return
	}
	a.startOnce.Do(func() { a.begin(t, skipped) })
}

func (a *attempt) begin(t *testing.T, skipped bool) {
	parent := a.cfg.Context.Raw
	if parent == nil {
		parent = context.Background()
	}
	attrs := []attribute.KeyValue{
		attribute.String(attributeTestName, t.Name()),
		attribute.Int(attributeExecutionAttempt, max(a.cfg.Execution.Attempt, 1)),
	}
	if a.cfg.Execution.ID != "" {
		attrs = append(attrs, attribute.String(attributeExecutionID, a.cfg.Execution.ID))
	}
	if a.cfg.Case != nil {
		attrs = append(attrs, attribute.String(attributeCaseName, a.cfg.Case.Name))
		if a.cfg.Case.ID != "" {
			attrs = append(attrs, attribute.String(attributeCaseID, a.cfg.Case.ID))
		}
	}

	ctx, span := a.tracer.Start(parent, testSpanName, trace.WithAttributes(attrs...))
	a.mu.Lock()
	a.ctx, a.span = ctx, span
	a.started = true
	a.skipped = skipped
	a.mu.Unlock()
	if skipped {
		span.AddEvent(string(axiom.EventTypeCaseSkip))
	}
	// This is registered before the attempt's hooks and body register their
	// cleanups, so it runs after them and includes their time and failures.
	t.Cleanup(func() { a.finish(t) })
}

func (a *attempt) record(event axiom.Event) {
	if !lifecycleEvent(event.Type) {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()
	if !a.started || a.ended {
		return
	}
	var options []trace.EventOption
	if event.Name != "" {
		options = append(options, trace.WithAttributes(attribute.String(attributeEventName, event.Name)))
	}
	a.span.AddEvent(string(event.Type), options...)
	if failureEvent(event.Type) {
		a.failed = true
		a.span.SetStatus(codes.Error, string(event.Type))
	}
}

func (a *attempt) finish(t *testing.T) {
	a.mu.Lock()
	if !a.started || a.ended {
		a.mu.Unlock()
		return
	}
	a.ended = true
	span := a.span
	failed := a.failed || t.Failed()
	skipped := a.skipped || t.Skipped()
	a.mu.Unlock()

	switch {
	case failed:
		span.SetAttributes(attribute.String(attributeTestStatus, statusFailed))
		span.SetStatus(codes.Error, "test failed")
	case skipped:
		span.SetAttributes(attribute.String(attributeTestStatus, statusSkipped))
	default:
		span.SetAttributes(attribute.String(attributeTestStatus, statusPassed))
	}
	span.End()
}

func lifecycleEvent(event axiom.EventType) bool {
	switch event {
	case axiom.EventTypeCasePanic,
		axiom.EventTypeStepStart, axiom.EventTypeStepFinish, axiom.EventTypeStepPanic,
		axiom.EventTypeSetupStart, axiom.EventTypeSetupFinish, axiom.EventTypeSetupPanic,
		axiom.EventTypeTeardownStart, axiom.EventTypeTeardownFinish, axiom.EventTypeTeardownPanic,
		axiom.EventTypeFixtureSetupStart, axiom.EventTypeFixtureSetupFinish, axiom.EventTypeFixtureSetupFailed,
		axiom.EventTypeFixtureCleanupStart, axiom.EventTypeFixtureCleanupFinish, axiom.EventTypeFixtureCleanupPanic:
		return true
	default:
		return false
	}
}

func failureEvent(event axiom.EventType) bool {
	switch event {
	case axiom.EventTypeCasePanic, axiom.EventTypeStepPanic,
		axiom.EventTypeSetupPanic, axiom.EventTypeTeardownPanic,
		axiom.EventTypeFixtureSetupFailed, axiom.EventTypeFixtureCleanupPanic:
		return true
	default:
		return false
	}
}
