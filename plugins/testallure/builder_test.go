package testallure_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
	"github.com/dailymotion/allure-go"
	"github.com/stretchr/testify/assert"
)

func TestBuildAllureOptions_AllFields(t *testing.T) {
	cfg := &axiom.Config{
		ID:   "ID123",
		Name: "MyTest",
		Meta: axiom.Meta{
			Tags:     []string{"fast", "api"},
			Epic:     "Epic1",
			Story:    "Story1",
			Layer:    "API",
			Feature:  "Feature1",
			Severity: axiom.SeverityCritical,
			Labels: map[string]string{
				"team":  "backend",
				"owner": "nikita",
			},
		},
	}

	opts := testallure.BuildAllureOptions(cfg)

	assert.GreaterOrEqual(t, len(opts), 8)

	assert.NotPanics(t, func() {
		allure.Test(t, opts...)
	})
}

func TestBuildAllureOptions_EmptyConfig(t *testing.T) {
	cfg := &axiom.Config{
		Meta: axiom.Meta{},
	}

	opts := testallure.BuildAllureOptions(cfg)

	assert.Equal(t, 0, len(opts))
}

func TestBuildAllureOptions_OnlyLabels(t *testing.T) {
	cfg := &axiom.Config{
		Meta: axiom.Meta{
			Labels: map[string]string{
				"a": "1",
				"b": "2",
			},
		},
	}

	opts := testallure.BuildAllureOptions(cfg)

	assert.Equal(t, 2, len(opts))

	assert.NotPanics(t, func() {
		allure.Test(t, opts...)
	})
}

func TestBuildAllureOptions_SeverityConversion(t *testing.T) {
	cfg := &axiom.Config{
		Meta: axiom.Meta{
			Severity: axiom.SeverityMinor,
		},
	}

	opts := testallure.BuildAllureOptions(cfg)

	assert.Equal(t, 1, len(opts))

	assert.NotPanics(t, func() {
		allure.Test(t, opts...)
	})
}
