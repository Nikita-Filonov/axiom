package axiom

import (
	"encoding/json"
)

type ArtefactType string

const (
	ArtefactTypeText  ArtefactType = "text"
	ArtefactTypeJSON  ArtefactType = "json"
	ArtefactTypeBytes ArtefactType = "bytes"
)

type Artefact struct {
	Name string
	Type ArtefactType
	Data []byte
}

type ArtefactOption func(*Artefact)

func NewArtefact(options ...ArtefactOption) Artefact {
	a := Artefact{}
	for _, option := range options {
		option(&a)
	}

	return a
}

func WithArtefactName(name string) ArtefactOption {
	return func(a *Artefact) { a.Name = name }
}

func WithArtefactType(t ArtefactType) ArtefactOption {
	return func(a *Artefact) { a.Type = t }
}

func WithArtefactData(data []byte) ArtefactOption {
	return func(a *Artefact) { a.Data = data }
}

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

func NewTextArtefact(name string, text string) Artefact {
	return NewArtefact(
		WithArtefactName(name),
		WithArtefactType(ArtefactTypeText),
		WithArtefactData([]byte(text)),
	)
}

func NewBytesArtefact(name string, data []byte) Artefact {
	return NewArtefact(
		WithArtefactName(name),
		WithArtefactType(ArtefactTypeBytes),
		WithArtefactData(data),
	)
}
