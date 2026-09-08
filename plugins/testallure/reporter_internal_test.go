package testallure

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Nikita-Filonov/axiom"
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
		"backend-integration-tests/tests.TestCreateAtm(...)",
		"\t/repo/tests/atm_test.go:42 +0x33",
		"testing.tRunner(...)",
		"\t/usr/local/go/src/testing/testing.go:1690 +0xf0",
		"",
	}, "\n")

	filtered := filterStack([]byte(raw), stackDropPrefixes)

	if !strings.HasPrefix(filtered, "goroutine 12 [running]:") {
		t.Fatalf("expected goroutine header to be kept, got:\n%s", filtered)
	}
	for _, dropped := range []string{
		"runtime/debug.Stack()",
		"(*TestingT).Errorf",
		"github.com/stretchr/testify/assert.Fail",
	} {
		if strings.Contains(filtered, dropped) {
			t.Fatalf("expected %q to be dropped, got:\n%s", dropped, filtered)
		}
	}
	for _, kept := range []string{
		"backend-integration-tests/tests.TestCreateAtm(...)",
		"/repo/tests/atm_test.go:42 +0x33",
		"testing.tRunner(...)",
	} {
		if !strings.Contains(filtered, kept) {
			t.Fatalf("expected %q to be kept, got:\n%s", kept, filtered)
		}
	}
}

func TestFilterStack_EmptyInput(t *testing.T) {
	if got := filterStack(nil, stackDropPrefixes); got != "" {
		t.Fatalf("expected empty result, got %q", got)
	}
}

func TestComposeMessage(t *testing.T) {
	t.Run("inline appends stack after message", func(t *testing.T) {
		got := composeMessage("Not equal", "goroutine 1\nframe", true)

		want := "Not equal\n\n" + stackSectionHeader + "\ngoroutine 1\nframe"
		if got != want {
			t.Fatalf("unexpected message:\ngot:  %q\nwant: %q", got, want)
		}
	})

	t.Run("inline disabled returns message only", func(t *testing.T) {
		if got := composeMessage("Not equal", "stack", false); got != "Not equal" {
			t.Fatalf("expected plain message, got %q", got)
		}
	})

	t.Run("empty stack returns message only", func(t *testing.T) {
		if got := composeMessage("Not equal", "", true); got != "Not equal" {
			t.Fatalf("expected plain message, got %q", got)
		}
	})
}

func TestTestingT_Errorf_WithReporterRecordsMessageAndAttachment(t *testing.T) {
	reporter := &fakeReporter{}
	fail := &fakeFailer{}
	adapter := newTestingT(fail, constantReporter(reporter), defaultTOptions())

	adapter.Errorf("values must match: %s", "boom")

	if len(reporter.messages) != 1 {
		t.Fatalf("expected one reported message, got %d", len(reporter.messages))
	}
	message := reporter.messages[0]
	if !strings.Contains(message, "values must match: boom") {
		t.Fatalf("expected assertion message, got:\n%s", message)
	}
	if !strings.Contains(message, stackSectionHeader) {
		t.Fatalf("expected inlined stack header, got:\n%s", message)
	}

	if len(reporter.attachments) != 1 {
		t.Fatalf("expected one attachment, got %d", len(reporter.attachments))
	}
	attachment := reporter.attachments[0]
	if attachment.name != defaultStackAttachmentName {
		t.Fatalf("unexpected attachment name %q", attachment.name)
	}
	if attachment.contentType != contentTypeText {
		t.Fatalf("unexpected attachment content type %q", attachment.contentType)
	}
	if len(attachment.content) == 0 {
		t.Fatal("expected non-empty stack attachment")
	}

	// The reporter forwards to the underlying *testing.T itself, so the adapter
	// must not double-report to the failer.
	if len(fail.messages) != 0 {
		t.Fatalf("expected no direct failer reports, got %v", fail.messages)
	}
	if fail.helperHits == 0 {
		t.Fatal("expected Helper to be called on the failer")
	}
}

func TestTestingT_Errorf_OptionsCanDisableInlineAndAttachment(t *testing.T) {
	reporter := &fakeReporter{}
	adapter := newTestingT(
		&fakeFailer{},
		constantReporter(reporter),
		tOptions{inlineStack: false, attachStack: false, attachmentName: defaultStackAttachmentName},
	)

	adapter.Errorf("plain failure")

	if len(reporter.messages) != 1 || reporter.messages[0] != "plain failure" {
		t.Fatalf("expected plain message without stack, got %v", reporter.messages)
	}
	if len(reporter.attachments) != 0 {
		t.Fatalf("expected no attachment, got %d", len(reporter.attachments))
	}
}

func TestTestingT_Errorf_WithoutReporterFallsBackToFailer(t *testing.T) {
	fail := &fakeFailer{}
	adapter := newTestingT(fail, constantReporter(nil), defaultTOptions())

	adapter.Errorf("fallback %d", 7)

	if len(fail.messages) != 1 || fail.messages[0] != "fallback 7" {
		t.Fatalf("expected fallback to failer, got %v", fail.messages)
	}
}

func TestTestingT_Errorf_NilCurrentFallsBackToFailer(t *testing.T) {
	fail := &fakeFailer{}
	adapter := newTestingT(fail, nil, defaultTOptions())

	adapter.Errorf("no current")

	if len(fail.messages) != 1 || fail.messages[0] != "no current" {
		t.Fatalf("expected fallback to failer, got %v", fail.messages)
	}
}

func TestTestingT_FailNowDelegatesToFailer(t *testing.T) {
	fail := &fakeFailer{}
	adapter := newTestingT(fail, constantReporter(&fakeReporter{}), defaultTOptions())

	adapter.FailNow()

	if fail.failNow != 1 {
		t.Fatalf("expected FailNow to delegate once, got %d", fail.failNow)
	}
}

func TestTestingT_FailNowWithoutFailerIsNoop(t *testing.T) {
	adapter := newTestingT(nil, constantReporter(&fakeReporter{}), defaultTOptions())

	// Must not panic when there is no underlying *testing.T.
	adapter.FailNow()
}

func TestTestingT_HelperDelegatesToFailer(t *testing.T) {
	fail := &fakeFailer{}
	adapter := newTestingT(fail, constantReporter(&fakeReporter{}), defaultTOptions())

	adapter.Helper()

	if fail.helperHits != 1 {
		t.Fatalf("expected Helper to delegate once, got %d", fail.helperHits)
	}
}

func TestTOptions(t *testing.T) {
	t.Run("defaults enable inline stack and attachment", func(t *testing.T) {
		opts := defaultTOptions()
		if !opts.inlineStack || !opts.attachStack || opts.attachmentName != defaultStackAttachmentName {
			t.Fatalf("unexpected defaults: %+v", opts)
		}
	})

	t.Run("WithInlineStack toggles inlineStack", func(t *testing.T) {
		opts := defaultTOptions()

		WithInlineStack(false)(&opts)
		if opts.inlineStack {
			t.Fatal("expected inlineStack disabled")
		}

		WithInlineStack(true)(&opts)
		if !opts.inlineStack {
			t.Fatal("expected inlineStack enabled")
		}
	})

	t.Run("WithStackAttachment toggles attachStack", func(t *testing.T) {
		opts := defaultTOptions()

		WithStackAttachment(false)(&opts)
		if opts.attachStack {
			t.Fatal("expected attachStack disabled")
		}

		WithStackAttachment(true)(&opts)
		if !opts.attachStack {
			t.Fatal("expected attachStack enabled")
		}
	})

	t.Run("WithStackAttachmentName overrides and ignores empty", func(t *testing.T) {
		opts := defaultTOptions()

		WithStackAttachmentName("")(&opts)
		if opts.attachmentName != defaultStackAttachmentName {
			t.Fatalf("expected default name preserved, got %q", opts.attachmentName)
		}

		WithStackAttachmentName("Trace")(&opts)
		if opts.attachmentName != "Trace" {
			t.Fatalf("expected overridden name, got %q", opts.attachmentName)
		}
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

	if adapter.opts.inlineStack || adapter.opts.attachStack {
		t.Fatalf("expected options applied, got %+v", adapter.opts)
	}
	if adapter.opts.attachmentName != "Trace" {
		t.Fatalf("unexpected attachment name %q", adapter.opts.attachmentName)
	}
	if adapter.fail == nil {
		t.Fatal("expected reporter to wrap cfg.T()")
	}
	if adapter.current == nil {
		t.Fatal("expected current resolver to be set")
	}
	if adapter.current() != nil {
		t.Fatal("expected nil reporter while no Allure context is active")
	}
}

func TestT_PanicsWhenPluginStateMissing(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic when plugin state is missing")
		}
	}()

	_ = T(&axiom.Config{SubT: t})
}
