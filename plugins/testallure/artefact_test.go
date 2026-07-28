package testallure_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlugin_UnsupportedArtefactDoesNothing(t *testing.T) {
	var logs []axiom.Log
	cfg := &axiom.Config{
		SubT: t,
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeLogSink(func(log axiom.Log) {
				logs = append(logs, log)
			}),
		),
	}
	testallure.Plugin()(cfg)

	cfg.Runtime.Artefact(axiom.Artefact{
		Name: "unsupported",
		Type: axiom.ArtefactType("unsupported"),
		Data: []byte("123"),
	})

	assert.Empty(t, logs)
}

func TestPlugin_SupportedArtefactOutsideTestLogsWarning(t *testing.T) {
	var logs []axiom.Log
	cfg := &axiom.Config{
		SubT: t,
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeLogSink(func(log axiom.Log) {
				logs = append(logs, log)
			}),
		),
	}
	testallure.Plugin()(cfg)

	cfg.Runtime.Artefact(axiom.Artefact{
		Name: "orphan.json",
		Type: axiom.ArtefactTypeJSON,
		Data: []byte(`{"ok":true}`),
	})

	require.Len(t, logs, 1)
	assert.Equal(t, axiom.LogLevelWarning, logs[0].Level)
	assert.Contains(t, logs[0].Text, "no active allure test context")
}
