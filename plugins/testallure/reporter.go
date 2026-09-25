package testallure

import (
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/Nikita-Filonov/axiom"
)

const (
	stackSectionHeader         = "--- Stack trace ---"
	defaultStackAttachmentName = "Stacktrace"
)

var stackDropPrefixes = []string{
	"runtime/debug.",
	"github.com/stretchr/testify/",
	"github.com/onsi/gomega/",
	"github.com/Nikita-Filonov/axiom/plugins/testallure.",
}

type allureReporter interface {
	Errorf(format string, args ...any)
	Attachment(name string, content []byte, contentType string)
}

type failer interface {
	Errorf(format string, args ...any)
	FailNow()
	Helper()
}

type tOptions struct {
	inlineStack    bool
	attachStack    bool
	attachmentName string
}

func defaultTOptions() tOptions {
	return tOptions{inlineStack: true, attachStack: true, attachmentName: defaultStackAttachmentName}
}

// TOption configures the assertion adapter returned by T.
type TOption func(*tOptions)

// WithInlineStack controls whether assertion errors include a stack trace in
// their message. It is enabled by default.
func WithInlineStack(enabled bool) TOption {
	return func(o *tOptions) { o.inlineStack = enabled }
}

// WithStackAttachment controls whether assertion errors attach a separate
// stack trace. It is enabled by default.
func WithStackAttachment(enabled bool) TOption {
	return func(o *tOptions) { o.attachStack = enabled }
}

// WithStackAttachmentName names the stack trace attachment. An empty name
// leaves the default name unchanged.
func WithStackAttachmentName(name string) TOption {
	return func(o *tOptions) {
		if name != "" {
			o.attachmentName = name
		}
	}
}

// TestingT adapts testing-style failures to the active Allure test context.
type TestingT struct {
	fail    failer
	current func() allureReporter
	opts    tOptions
}

// T returns a testing-style adapter for the active Axiom test. Plugin must
// have been applied to cfg first.
func T(cfg *axiom.Config, options ...TOption) *TestingT {
	opts := defaultTOptions()
	for _, option := range options {
		option(&opts)
	}

	state := axiom.MustContextValue[*allureContextState](&cfg.Context, contextStateKey)
	current := func() allureReporter {
		if ctx := state.current.Load(); ctx != nil {
			return ctx
		}
		return nil
	}

	return newTestingT(cfg.T(), current, opts)
}

func newTestingT(fail failer, current func() allureReporter, opts tOptions) *TestingT {
	return &TestingT{fail: fail, current: current, opts: opts}
}

// Errorf records a non-fatal assertion failure and its optional stack trace.
func (a *TestingT) Errorf(format string, args ...any) {
	if a.fail != nil {
		a.fail.Helper()
	}

	message := fmt.Sprintf(format, args...)
	reporter := a.reporter()
	if reporter == nil {
		if a.fail != nil {
			a.fail.Errorf("%s", message)
		}
		return
	}

	stack := filterStack(debug.Stack(), stackDropPrefixes)
	reporter.Errorf("%s", composeMessage(message, stack, a.opts.inlineStack))
	if a.opts.attachStack && stack != "" {
		reporter.Attachment(a.opts.attachmentName, []byte(stack), contentTypeText)
	}
}

// Fatalf records a failure and stops the current test goroutine.
func (a *TestingT) Fatalf(format string, args ...any) {
	a.Errorf(format, args...)
	a.FailNow()
}

// FailNow stops the current test goroutine.
func (a *TestingT) FailNow() {
	if a.fail == nil {
		return
	}
	a.fail.Helper()
	a.fail.FailNow()
}

// Helper marks the caller as a test helper.
func (a *TestingT) Helper() {
	if a.fail != nil {
		a.fail.Helper()
	}
}

func (a *TestingT) reporter() allureReporter {
	if a.current == nil {
		return nil
	}
	return a.current()
}

func composeMessage(message, stack string, inline bool) string {
	if !inline || stack == "" {
		return message
	}

	var b strings.Builder
	b.WriteString(message)
	if message != "" && !strings.HasSuffix(message, "\n") {
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(stackSectionHeader)
	b.WriteByte('\n')
	b.WriteString(stack)

	return b.String()
}

func filterStack(stack []byte, dropPrefixes []string) string {
	lines := strings.Split(strings.TrimRight(string(stack), "\n"), "\n")

	result := make([]string, 0, len(lines))
	start := 0
	if strings.HasPrefix(lines[0], "goroutine ") {
		result = append(result, lines[0])
		start = 1
	}

	for i := start; i < len(lines); i += 2 {
		if hasAnyPrefix(lines[i], dropPrefixes) {
			continue
		}
		result = append(result, lines[i])
		if i+1 < len(lines) {
			result = append(result, lines[i+1])
		}
	}

	return strings.Join(result, "\n")
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
