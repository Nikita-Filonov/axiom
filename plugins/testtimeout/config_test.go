package testtimeout_test

import (
	"testing"
	"time"

	"github.com/Nikita-Filonov/axiom/plugins/testtimeout"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig_Defaults(t *testing.T) {
	cfg := testtimeout.NewConfig()

	assert.Equal(t, time.Duration(0), cfg.Timeout)
	assert.True(t, cfg.DumpGoroutines)
	assert.Empty(t, cfg.Message)
}

func TestWithTimeout(t *testing.T) {
	cfg := testtimeout.NewConfig(testtimeout.WithTimeout(3 * time.Second))

	assert.Equal(t, 3*time.Second, cfg.Timeout)
}

func TestWithoutGoroutineDump(t *testing.T) {
	cfg := testtimeout.NewConfig(testtimeout.WithoutGoroutineDump())

	assert.False(t, cfg.DumpGoroutines)
}

func TestWithMessage(t *testing.T) {
	cfg := testtimeout.NewConfig(testtimeout.WithMessage("too slow"))

	assert.Equal(t, "too slow", cfg.Message)
}

func TestConfigFromEnv_Valid(t *testing.T) {
	t.Setenv(testtimeout.AxiomTestTimeout, "250ms")

	cfg := testtimeout.NewConfig(testtimeout.ConfigFromEnv())

	assert.Equal(t, 250*time.Millisecond, cfg.Timeout)
}

func TestConfigFromEnv_Empty(t *testing.T) {
	t.Setenv(testtimeout.AxiomTestTimeout, "")

	cfg := testtimeout.NewConfig(testtimeout.WithTimeout(time.Second), testtimeout.ConfigFromEnv())

	assert.Equal(t, time.Second, cfg.Timeout, "an empty value must not override the current timeout")
}

func TestConfigFromEnv_Invalid(t *testing.T) {
	t.Setenv(testtimeout.AxiomTestTimeout, "not-a-duration")

	cfg := testtimeout.NewConfig(testtimeout.WithTimeout(time.Second), testtimeout.ConfigFromEnv())

	assert.Equal(t, time.Second, cfg.Timeout, "an invalid value must not override the current timeout")
}
