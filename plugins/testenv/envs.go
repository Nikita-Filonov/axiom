package testenv

import "strings"

// Envs is a read-only snapshot of environment variables. Values may contain
// secrets; do not log the snapshot or its values without checking their use.
type Envs struct {
	values map[string]string
}

// Lookup returns the raw value and whether the variable was explicitly set.
// An empty value with ok == true is different from an unset variable.
func (e *Envs) Lookup(name string) (value string, ok bool) {
	value, ok = e.values[name]
	return value, ok
}

func snapshot(entries []string) *Envs {
	values := make(map[string]string)
	for _, entry := range entries {
		name, value, ok := strings.Cut(entry, "=")
		if ok {
			values[name] = value
		}
	}
	return &Envs{values: values}
}
