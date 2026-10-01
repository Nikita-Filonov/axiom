package testallure

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	allure "github.com/allure-framework/allure-go/commons/gotest"
	"github.com/stretchr/testify/require"
)

// gomegaTestingT checks compatibility with Gomega without adding a dependency.
type gomegaTestingT interface {
	Helper()
	Fatalf(format string, args ...any)
}

var (
	_ gomegaTestingT   = (*TestingT)(nil)
	_ require.TestingT = (*TestingT)(nil)
)

type fakeReporter struct {
	messages    []string
	attachments []fakeAttachment
}

type fakeAttachment struct {
	name        string
	content     []byte
	contentType string
}

func (f *fakeReporter) Errorf(format string, args ...any) {
	f.messages = append(f.messages, fmt.Sprintf(format, args...))
}

func (f *fakeReporter) Attachment(name string, content []byte, contentType string) {
	f.attachments = append(f.attachments, fakeAttachment{
		name:        name,
		content:     content,
		contentType: contentType,
	})
}

type fakeFailer struct {
	messages   []string
	failNow    int
	helperHits int
}

func (f *fakeFailer) Errorf(format string, args ...any) {
	f.messages = append(f.messages, fmt.Sprintf(format, args...))
}

func (f *fakeFailer) FailNow() { f.failNow++ }

func (f *fakeFailer) Helper() { f.helperHits++ }

func constantReporter(reporter allureReporter) func() allureReporter {
	return func() allureReporter { return reporter }
}

func TestFilterStack_DropsNoiseFramesKeepsHeaderAndUserCode(t *testing.T) {
	raw := strings.Join([]string{
		"goroutine 12 [running]:",
		"runtime/debug.Stack()",
		"\t/usr/local/go/src/runtime/debug/stack.go:26 +0x5e",
		"github.com/Nikita-Filonov/axiom/plugins/testallure.(*TestingT).Errorf(...)",
		"\t/repo/plugins/testallure/reporter.go:140 +0x11",
		"github.com/stretchr/testify/assert.Fail(...)",
		"\t/repo/testify/assert/assertions.go:100 +0x22",
		"github.com/onsi/gomega/internal.(*AsyncAssertion).match(...)",
		"\t/repo/gomega/internal/async_assertion.go:200 +0x44",
		"backend-integration-tests/tests.TestCreateAtm(...)",
		"\t/repo/tests/atm_test.go:42 +0x33",
		"testing.tRunner(...)",
		"\t/usr/local/go/src/testing/testing.go:1690 +0xf0",
		"",
	}, "\n")

	filtered := filterStack([]byte(raw), stackDropPrefixes)

	require.True(t, strings.HasPrefix(filtered, "goroutine 12 [running]:"))
	for _, dropped := range []string{
		"runtime/debug.Stack()",
		"(*TestingT).Errorf",
		"github.com/stretchr/testify/assert.Fail",
		"github.com/onsi/gomega/internal.(*AsyncAssertion).match",
	} {
		require.NotContains(t, filtered, dropped)
	}
	for _, kept := range []string{
		"backend-integration-tests/tests.TestCreateAtm(...)",
		"/repo/tests/atm_test.go:42 +0x33",
		"testing.tRunner(...)",
	} {
		require.Contains(t, filtered, kept)
	}
}

func TestFilterStack_EmptyInput(t *testing.T) {
	require.Empty(t, filterStack(nil, stackDropPrefixes))
}

func TestComposeMessage(t *testing.T) {
	t.Run("inline appends stack after message", func(t *testing.T) {
		got := composeMessage("Not equal", "goroutine 1\nframe", true)

		want := "Not equal\n\n" + stackSectionHeader + "\ngoroutine 1\nframe"
		require.Equal(t, want, got)
	})

	t.Run("inline disabled returns message only", func(t *testing.T) {
		require.Equal(t, "Not equal", composeMessage("Not equal", "stack", false))
	})

	t.Run("empty stack returns message only", func(t *testing.T) {
		require.Equal(t, "Not equal", composeMessage("Not equal", "", true))
	})
}

func TestTestingT_Errorf_WithReporterRecordsMessageAndAttachment(t *testing.T) {
	reporter := &fakeReporter{}
	fail := &fakeFailer{}
	adapter := newTestingT(fail, constantReporter(reporter), defaultTOptions())

	adapter.Errorf("values must match: %s", "boom")

	require.Len(t, reporter.messages, 1)
	message := reporter.messages[0]
	require.Contains(t, message, "values must match: boom")
	require.Contains(t, message, stackSectionHeader)

	require.Len(t, reporter.attachments, 1)
	attachment := reporter.attachments[0]
	require.Equal(t, defaultStackAttachmentName, attachment.name)
	require.Equal(t, contentTypeText, attachment.contentType)
	require.NotEmpty(t, attachment.content)

	// The reporter forwards to the underlying *testing.T itself, so the adapter
	// must not double-report to the failer.
	require.Empty(t, fail.messages)
	require.NotEqual(t, 0, fail.helperHits)
}

func TestTestingT_Errorf_OptionsCanDisableInlineAndAttachment(t *testing.T) {
	reporter := &fakeReporter{}
	adapter := newTestingT(
		&fakeFailer{},
		constantReporter(reporter),
		tOptions{inlineStack: false, attachStack: false, attachmentName: defaultStackAttachmentName},
	)

	adapter.Errorf("plain failure")

	require.Len(t, reporter.messages, 1)
	require.Equal(t, "plain failure", reporter.messages[0])
	require.Empty(t, reporter.attachments)
}

func TestTestingT_Errorf_WithoutReporterFallsBackToFailer(t *testing.T) {
	fail := &fakeFailer{}
	adapter := newTestingT(fail, constantReporter(nil), defaultTOptions())

	adapter.Errorf("fallback %d", 7)

	require.Len(t, fail.messages, 1)
	require.Equal(t, "fallback 7", fail.messages[0])
}

func TestTestingT_Errorf_NilCurrentFallsBackToFailer(t *testing.T) {
	fail := &fakeFailer{}
	adapter := newTestingT(fail, nil, defaultTOptions())

	adapter.Errorf("no current")

	require.Len(t, fail.messages, 1)
	require.Equal(t, "no current", fail.messages[0])
}

func TestTestingT_FailNowDelegatesToFailer(t *testing.T) {
	fail := &fakeFailer{}
	adapter := newTestingT(fail, constantReporter(&fakeReporter{}), defaultTOptions())

	adapter.FailNow()

	require.Equal(t, 1, fail.failNow)
}

func TestTestingT_FailNowWithoutFailerIsNoop(t *testing.T) {
	adapter := newTestingT(nil, constantReporter(&fakeReporter{}), defaultTOptions())

	// Must not panic when there is no underlying *testing.T.
	adapter.FailNow()
}

func TestTestingT_Fatalf_RecordsMessageAndStopsTest(t *testing.T) {
	reporter := &fakeReporter{}
	fail := &fakeFailer{}
	adapter := newTestingT(fail, constantReporter(reporter), defaultTOptions())

	// gomega.NewWithT invokes the reporter as t.Fatalf("\n%s", message).
	adapter.Fatalf("\n%s", "Timed out after 60s.\nExpected\n    <bool>: false\nto be true")

	require.Len(t, reporter.messages, 1)
	require.Contains(t, reporter.messages[0], "Timed out after 60s.")
	require.Contains(t, reporter.messages[0], stackSectionHeader)
	require.Len(t, reporter.attachments, 1)
	require.Equal(t, 1, fail.failNow)
}

func TestTestingT_Fatalf_WithoutReporterFallsBackToFailer(t *testing.T) {
	fail := &fakeFailer{}
	adapter := newTestingT(fail, constantReporter(nil), defaultTOptions())

	adapter.Fatalf("boom %d", 7)

	require.Len(t, fail.messages, 1)
	require.Equal(t, "boom 7", fail.messages[0])
	require.Equal(t, 1, fail.failNow)
}

func TestTestingT_HelperDelegatesToFailer(t *testing.T) {
	fail := &fakeFailer{}
	adapter := newTestingT(fail, constantReporter(&fakeReporter{}), defaultTOptions())

	adapter.Helper()

	require.Equal(t, 1, fail.helperHits)
}

func TestTOptions(t *testing.T) {
	t.Run("defaults enable inline stack and attachment", func(t *testing.T) {
		opts := defaultTOptions()
		require.True(t, opts.inlineStack)
		require.True(t, opts.attachStack)
		require.Equal(t, defaultStackAttachmentName, opts.attachmentName)
	})

	t.Run("WithInlineStack toggles inlineStack", func(t *testing.T) {
		opts := defaultTOptions()

		WithInlineStack(false)(&opts)
		require.False(t, opts.inlineStack)

		WithInlineStack(true)(&opts)
		require.True(t, opts.inlineStack)
	})

	t.Run("WithStackAttachment toggles attachStack", func(t *testing.T) {
		opts := defaultTOptions()

		WithStackAttachment(false)(&opts)
		require.False(t, opts.attachStack)

		WithStackAttachment(true)(&opts)
		require.True(t, opts.attachStack)
	})

	t.Run("WithStackAttachmentName overrides and ignores empty", func(t *testing.T) {
		opts := defaultTOptions()

		WithStackAttachmentName("")(&opts)
		require.Equal(t, defaultStackAttachmentName, opts.attachmentName)

		WithStackAttachmentName("Trace")(&opts)
		require.Equal(t, "Trace", opts.attachmentName)
	})
}

func TestT_AppliesOptionsAndWrapsConfigT(t *testing.T) {
	cfg := &axiom.Config{SubT: t}
	cfg.Context.SetData(contextStateKey, &allureContextState{})

	adapter := T(cfg,
		WithInlineStack(false),
		WithStackAttachment(false),
		WithStackAttachmentName("Trace"),
	)

	require.False(t, adapter.opts.inlineStack)
	require.False(t, adapter.opts.attachStack)
	require.Equal(t, "Trace", adapter.opts.attachmentName)
	require.NotNil(t, adapter.fail)
	require.NotNil(t, adapter.current)
	require.Nil(t, adapter.current())
}

func TestT_ResolvesActiveAllureContext(t *testing.T) {
	active := new(allure.Context)
	state := &allureContextState{}
	state.current.Store(active)
	cfg := &axiom.Config{SubT: t}
	cfg.Context.SetData(contextStateKey, state)

	adapter := T(cfg)
	require.Same(t, active, adapter.current())
}

func TestT_PanicsWhenPluginStateMissing(t *testing.T) {
	require.Panics(t, func() { T(&axiom.Config{SubT: t}) })
}
