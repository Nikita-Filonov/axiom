package testallure

import (
	"github.com/Nikita-Filonov/axiom"
	"github.com/dailymotion/allure-go"
)

func HandleArtefact(cfg *axiom.Config, a axiom.Artefact) {
	var err error
	switch a.Type {
	case axiom.ArtefactTypeJSON:
		err = allure.AddAttachment(a.Name, allure.ApplicationJson, a.Data)
	case axiom.ArtefactTypeText:
		err = allure.AddAttachment(a.Name, allure.TextPlain, a.Data)
	}

	if err != nil {
		cfg.Log(axiom.Log{
			Level: axiom.LogLevelWarning,
			Text:  "failed to add allure attachment: " + err.Error(),
		})
	}
}
