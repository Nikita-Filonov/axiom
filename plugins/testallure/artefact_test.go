package testallure_test

import (
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testallure"
	"github.com/stretchr/testify/assert"
)

func TestHandleArtefact_UnsupportedType_DoesNothing(t *testing.T) {
	var logs []axiom.Log

	cfg := &axiom.Config{
		Runtime: axiom.NewRuntime(
			axiom.WithRuntimeLogSink(func(l axiom.Log) {
				logs = append(logs, l)
			}),
		),
	}

	a := axiom.Artefact{
		Name: "bad",
		Type: axiom.ArtefactTypeBytes,
		Data: []byte("123"),
	}

	testallure.HandleArtefact(cfg, a)

	assert.Len(t, logs, 0, "no logs expected for unsupported artefact type")
}
