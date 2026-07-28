package testallure

import (
	"github.com/Nikita-Filonov/axiom"
	allure "github.com/allure-framework/allure-go/commons/gotest"
)

const (
	contentTypeBytes = "application/octet-stream"
	contentTypeJSON  = "application/json"
	contentTypeText  = "text/plain"
)

func handleArtefact(ctx *allure.Context, cfg *axiom.Config, a axiom.Artefact) {
	var contentType string
	switch a.Type {
	case axiom.ArtefactTypeBytes:
		contentType = contentTypeBytes
	case axiom.ArtefactTypeJSON:
		contentType = contentTypeJSON
	case axiom.ArtefactTypeText:
		contentType = contentTypeText
	default:
		return
	}

	if ctx == nil {
		if cfg != nil {
			cfg.Log(axiom.Log{
				Level: axiom.LogLevelWarning,
				Text:  "failed to add allure attachment: no active allure test context",
			})
		}
		return
	}

	ctx.Attachment(a.Name, a.Data, contentType)
}
