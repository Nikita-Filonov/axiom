package testenv_test

import (
	"fmt"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testenv"
)

func ExampleString() {
	target := testenv.String(
		testenv.WithName("TEST_ENV"),
		testenv.WithDefault("local"),
	)
	runner := axiom.NewRunner(
		axiom.WithRunnerResources(
			testenv.Resource(
				testenv.WithSource(func() []string { return []string{"TEST_ENV=stable"} }),
			),
		),
	)
	fmt.Println(target.Get(runner))
	// Output: stable
}
