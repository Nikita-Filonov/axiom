package usage_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/docs/usage"
)

func TestAdminUserBan(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseName("admin can ban user"),
	)

	usage.UsersAdminRunner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Step("ban user", func() {
			fmt.Println("user banned")
		})
	})
}
