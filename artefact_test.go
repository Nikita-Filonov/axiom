package axiom_test

import (
	"encoding/json"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewArtefact_WithOptions(t *testing.T) {
	data := []byte("payload")

	a := axiom.NewArtefact(
		axiom.WithArtefactName("test"),
		axiom.WithArtefactType(axiom.ArtefactTypeText),
		axiom.WithArtefactData(data),
	)

	assert.Equal(t, "test", a.Name)
	assert.Equal(t, axiom.ArtefactTypeText, a.Type)
	assert.Equal(t, data, a.Data)
}

func TestNewTextArtefact(t *testing.T) {
	a := axiom.NewTextArtefact("log", "hello world")

	assert.Equal(t, "log", a.Name)
	assert.Equal(t, axiom.ArtefactTypeText, a.Type)
	assert.Equal(t, []byte("hello world"), a.Data)
}

func TestNewBytesArtefact(t *testing.T) {
	raw := []byte{1, 2, 3}

	a := axiom.NewBytesArtefact("bin", raw)

	assert.Equal(t, "bin", a.Name)
	assert.Equal(t, axiom.ArtefactTypeBytes, a.Type)
	assert.Equal(t, raw, a.Data)
}

func TestNewJSONArtefact_OK(t *testing.T) {
	input := map[string]any{
		"id":   1,
		"name": "test",
	}

	a, err := axiom.NewJSONArtefact("json", input)
	require.NoError(t, err)

	assert.Equal(t, "json", a.Name)
	assert.Equal(t, axiom.ArtefactTypeJSON, a.Type)
	assert.NotEmpty(t, a.Data)

	var decoded map[string]any
	err = json.Unmarshal(a.Data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, float64(1), decoded["id"]) // json.Unmarshal -> float64
	assert.Equal(t, "test", decoded["name"])
}

func TestNewJSONArtefact_Error(t *testing.T) {
	input := func() {}

	a, err := axiom.NewJSONArtefact("broken", input)

	require.Error(t, err)
	assert.Equal(t, axiom.Artefact{}, a)
}

func TestNewArtefact_Empty(t *testing.T) {
	a := axiom.NewArtefact()

	assert.Empty(t, a.Name)
	assert.Empty(t, a.Type)
	assert.Nil(t, a.Data)
}

func TestNewJSONArtefact_Indented(t *testing.T) {
	a, err := axiom.NewJSONArtefact("x", map[string]int{"a": 1})
	require.NoError(t, err)

	assert.Contains(t, string(a.Data), "\n")
	assert.Contains(t, string(a.Data), "  ")
}
