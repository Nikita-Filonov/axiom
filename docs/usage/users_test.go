package usage_test

import (
	"fmt"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/docs/usage"
)

func TestUserLogin(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseName("user can login"),
		axiom.WithCaseMeta(
			axiom.WithMetaStory("login"),
			axiom.WithMetaTag("smoke"),
		),
		axiom.WithCaseFixture("user", usage.UserFixture),
	)

	usage.UsersRunner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Step("prepare user", func() {
			user := axiom.GetFixture[string](cfg, "user")
			fmt.Println("using:", user)
		})

		cfg.Step("login", func() {
			fmt.Println("login OK")
		})
	})
}

func TestUserLogin_InvalidPassword(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseName("user cannot login with invalid password"),
		axiom.WithCaseMeta(
			axiom.WithMetaStory("login"),
			axiom.WithMetaTag("negative"),
		),
		axiom.WithCaseFixture("user", usage.UserFixture),
	)

	usage.UsersRunner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Step("prepare user", func() {
			user := axiom.GetFixture[string](cfg, "user")
			fmt.Println("using:", user)
		})

		cfg.Step("attempt login", func() {
			fmt.Println("login failed: invalid password")
		})
	})
}

func TestUserLogin_FlakyBackend(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseName("user login with flaky backend"),
		axiom.WithCaseMeta(
			axiom.WithMetaStory("login"),
			axiom.WithMetaTag("flaky"),
		),
		axiom.WithCaseRetry(
			axiom.WithRetryTimes(5),
		),
		axiom.WithCaseFixture("user", usage.UserFixture),
	)

	usage.UsersRunner.RunCase(t, c, func(cfg *axiom.Config) {
		cfg.Step("login", func() {
			fmt.Println("backend timeout, retrying...")
		})
	})
}

type LoginParams struct {
	Username string
	Password string
}

func TestUserLogin_WithParams(t *testing.T) {
	c := axiom.NewCase(
		axiom.WithCaseName("user login with params"),
		axiom.WithCaseParams(LoginParams{
			Username: "john",
			Password: "secret",
		}),
	)

	usage.UsersRunner.RunCase(t, c, func(cfg *axiom.Config) {
		params := axiom.GetParams[LoginParams](cfg)

		cfg.Step("login", func() {
			fmt.Printf("login user=%s password=%s\n", params.Username, params.Password)
		})
	})
}
