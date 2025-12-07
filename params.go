package axiom

import (
	"fmt"
)

func GetParams[T any](cfg *Config) T {
	v, ok := cfg.Params.(T)
	if !ok {
		panic(fmt.Sprintf("engine: expected params of type %T but got %T", *new(T), cfg.Params))
	}
	return v
}
