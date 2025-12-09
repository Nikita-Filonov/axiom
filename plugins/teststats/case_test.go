package teststats_test

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/teststats"
	"github.com/stretchr/testify/assert"
)

func TestCaseResult_Finalize_Skipped(t *testing.T) {
	cfg := &axiom.Config{
		ID:   "1",
		Name: "TestSkip",
		Skip: axiom.Skip{Enabled: true},
	}

	cr := teststats.NewCaseResult(cfg)

	cr.Finalize(cfg, 1)

	assert.Equal(t, teststats.StatusSkipped, cr.Status)
	assert.Equal(t, 1, cr.Attempts)
	assert.Greater(t, cr.Duration, time.Duration(0))
}

func TestCaseResult_Finalize_Passed_SingleAttempt(t *testing.T) {
	cfg := &axiom.Config{
		ID:   "2",
		Name: "TestPass",
		SubT: t,
		Skip: axiom.Skip{},
	}

	cr := teststats.NewCaseResult(cfg)

	cr.Finalize(cfg, 1)

	assert.Equal(t, teststats.StatusPassed, cr.Status)
	assert.Equal(t, 1, cr.Attempts)
}

func TestCaseResult_Finalize_Flaky_MultipleAttempts(t *testing.T) {
	cfg := &axiom.Config{
		ID:   "3",
		Name: "TestFlaky",
		SubT: t,
	}

	cr := teststats.NewCaseResult(cfg)

	cr.Finalize(cfg, 3)

	assert.Equal(t, teststats.StatusFlaky, cr.Status)
	assert.Equal(t, 3, cr.Attempts)
}

func TestCaseResult_StoresMetaAndFields(t *testing.T) {
	cfg := &axiom.Config{
		ID:   "55",
		Name: "MetaCheck",
		Meta: axiom.Meta{
			Epic:  "E1",
			Story: "S1",
		},
		SubT: t,
	}

	cr := teststats.NewCaseResult(cfg)

	assert.Equal(t, "55", cr.ID)
	assert.Equal(t, "MetaCheck", cr.Name)
	assert.Equal(t, cfg.Meta, cr.Meta)
	assert.False(t, cr.Start.IsZero())
}
