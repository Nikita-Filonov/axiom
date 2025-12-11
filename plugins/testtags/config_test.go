package testtags_test

import (
	"os"
	"testing"

	"github.com/Nikita-Filonov/axiom/plugins/testtags"
	"github.com/stretchr/testify/assert"
)

func TestNewConfig_Defaults(t *testing.T) {
	cfg := testtags.NewConfig()

	assert.Empty(t, cfg.Include)
	assert.Empty(t, cfg.Exclude)
}

func TestWithConfigInclude_Normalizes(t *testing.T) {
	cfg := testtags.NewConfig(
		testtags.WithConfigInclude(" FAST ", "Api", "db"),
	)

	assert.Equal(t, []string{"fast", "api", "db"}, cfg.Include)
	assert.Empty(t, cfg.Exclude)
}

func TestWithConfigExclude_Normalizes(t *testing.T) {
	cfg := testtags.NewConfig(
		testtags.WithConfigExclude(" slow ", "   UI "),
	)

	assert.Equal(t, []string{"slow", "ui"}, cfg.Exclude)
	assert.Empty(t, cfg.Include)
}

func TestWithConfigInclude_MultipleCalls(t *testing.T) {
	cfg := testtags.NewConfig(
		testtags.WithConfigInclude("a"),
		testtags.WithConfigInclude("b", "C"),
	)

	assert.Equal(t, []string{"a", "b", "c"}, cfg.Include)
}

func TestWithConfigExclude_MultipleCalls(t *testing.T) {
	cfg := testtags.NewConfig(
		testtags.WithConfigExclude("x"),
		testtags.WithConfigExclude("Y", "  Z "),
	)

	assert.Equal(t, []string{"x", "y", "z"}, cfg.Exclude)
}

func TestConfigFromEnv_ParsesIncludeExclude(t *testing.T) {
	// Backup original env
	oldInclude := os.Getenv(testtags.AxiomTestTagsInclude)
	oldExclude := os.Getenv(testtags.AxiomTestTagsExclude)
	defer func() {
		os.Setenv(testtags.AxiomTestTagsInclude, oldInclude)
		os.Setenv(testtags.AxiomTestTagsExclude, oldExclude)
	}()

	os.Setenv(testtags.AxiomTestTagsInclude, "fast, api , DB ")
	os.Setenv(testtags.AxiomTestTagsExclude, "slow,  ui")

	cfg := testtags.NewConfig(
		testtags.ConfigFromEnv(),
	)

	assert.Equal(t, []string{"fast", "api", "db"}, cfg.Include)
	assert.Equal(t, []string{"slow", "ui"}, cfg.Exclude)
}

func TestConfigFromEnv_EmptyValues(t *testing.T) {
	// Backup original env
	oldInclude := os.Getenv(testtags.AxiomTestTagsInclude)
	oldExclude := os.Getenv(testtags.AxiomTestTagsExclude)
	defer func() {
		os.Setenv(testtags.AxiomTestTagsInclude, oldInclude)
		os.Setenv(testtags.AxiomTestTagsExclude, oldExclude)
	}()

	os.Setenv(testtags.AxiomTestTagsInclude, "")
	os.Setenv(testtags.AxiomTestTagsExclude, "")

	cfg := testtags.NewConfig(
		testtags.ConfigFromEnv(),
	)

	assert.Empty(t, cfg.Include)
	assert.Empty(t, cfg.Exclude)
}

func TestConfig_CombinedOptions(t *testing.T) {
	os.Setenv(testtags.AxiomTestTagsInclude, "net")
	defer os.Unsetenv(testtags.AxiomTestTagsInclude)

	cfg := testtags.NewConfig(
		testtags.WithConfigInclude("api"),
		testtags.ConfigFromEnv(),
		testtags.WithConfigExclude("slow"),
	)

	assert.Equal(t, []string{"api", "net"}, cfg.Include)
	assert.Equal(t, []string{"slow"}, cfg.Exclude)
}
