package teststats

import "github.com/Nikita-Filonov/axiom"

var installedKey = axiom.NewLocalKey[*recorder]("github.com/Nikita-Filonov/axiom/plugins/teststats.installed")

func installRecorder(cfg *axiom.Config) *recorder {
	if r, ok := axiom.GetLocal(cfg, installedKey); ok {
		return r
	}
	r := &recorder{
		cfg:        cfg,
		collectors: make(map[*Stats]*Attempt),
	}
	axiom.SetLocal(cfg, installedKey, r)
	cfg.Runtime.EmitEventSink(r.observe)
	return r
}
