package axiom

import (
	"encoding/json"
)

// ArtefactType identifies the format of an Artefact's data.
type ArtefactType string

// ArtefactType values describe the format sent to artefact sinks.
const (
	ArtefactTypeText  ArtefactType = "text"
	ArtefactTypeJSON  ArtefactType = "json"
	ArtefactTypeBytes ArtefactType = "bytes"
)

// String returns the artefact format name.
func (t ArtefactType) String() string {
	return string(t)
}

// Artefact carries named output bytes to runtime artefact sinks. Axiom does
// not persist the data; a configured sink decides how to store or report it.
type Artefact struct {
	Name string
	Type ArtefactType
	Data []byte
}

// ArtefactOption configures an Artefact.
type ArtefactOption func(*Artefact)

// NewArtefact returns an Artefact with the supplied options.
func NewArtefact(options ...ArtefactOption) Artefact {
	a := Artefact{}
	for _, option := range options {
		option(&a)
	}

	return a
}

// WithArtefactName sets the name shown to artefact sinks.
func WithArtefactName(name string) ArtefactOption {
	return func(a *Artefact) { a.Name = name }
}

// WithArtefactType identifies the format of the artefact data.
func WithArtefactType(t ArtefactType) ArtefactOption {
	return func(a *Artefact) { a.Type = t }
}

// WithArtefactData assigns data without copying its bytes.
func WithArtefactData(data []byte) ArtefactOption {
	return func(a *Artefact) { a.Data = data }
}

// NewJSONArtefact marshals v as indented JSON. It returns a marshal error
// without an Artefact when v cannot be encoded.
func NewJSONArtefact(name string, v any) (Artefact, error) {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return Artefact{}, err
	}

	return NewArtefact(
		WithArtefactName(name),
		WithArtefactType(ArtefactTypeJSON),
		WithArtefactData(data),
	), nil
}

// NewTextArtefact returns an Artefact containing the bytes of text.
func NewTextArtefact(name string, text string) Artefact {
	return NewArtefact(
		WithArtefactName(name),
		WithArtefactType(ArtefactTypeText),
		WithArtefactData([]byte(text)),
	)
}

// NewBytesArtefact returns an Artefact holding data without copying it.
func NewBytesArtefact(name string, data []byte) Artefact {
	return NewArtefact(
		WithArtefactName(name),
		WithArtefactType(ArtefactTypeBytes),
		WithArtefactData(data),
	)
}
