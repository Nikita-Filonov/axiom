package usage

import (
	"fmt"

	"github.com/Nikita-Filonov/axiom"
)

func ConfigFixture(cfg *axiom.Config) (any, func(), error) {
	env, _ := axiom.GetContextValue[string](&cfg.Context, "env")

	config := map[string]any{
		"env": env,
		"url": fmt.Sprintf("https://%s.example.com", env),
	}

	cleanup := func() {
		fmt.Println("cleanup config")
	}

	return config, cleanup, nil
}

func UserFixture(cfg *axiom.Config) (any, func(), error) {
	config := axiom.GetFixture[map[string]any](cfg, "config")

	user := fmt.Sprintf("user@%s", config["env"])

	cleanup := func() {
		fmt.Println("cleanup user:", user)
	}

	return user, cleanup, nil
}
