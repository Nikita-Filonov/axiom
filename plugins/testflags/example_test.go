package testflags_test

import (
	"flag"
	"fmt"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testflags"
)

func ExampleInt() {
	fs := flag.NewFlagSet("example", flag.ContinueOnError)
	custom := testflags.Int(
		testflags.WithName("axiom.custom"),
		testflags.WithDefault(0),
		testflags.WithUsage("custom value"),
		testflags.WithFlagSet(fs),
	)
	runner := axiom.NewRunner(
		axiom.WithRunnerResources(
			testflags.Resource(testflags.WithSource(fs)),
		),
	)
	if err := fs.Parse([]string{"-axiom.custom=55"}); err != nil {
		panic(err)
	}
	fmt.Println(custom.Get(runner))
	entry, _ := testflags.Get(runner).Lookup("axiom.custom")
	fmt.Println(entry.Default, entry.Set)
	// Output:
	// 55
	// 0 true
}

func ExampleResource() {
	fs := flag.NewFlagSet("example", flag.ContinueOnError)
	endpoint := testflags.String(
		testflags.WithName("api.url"),
		testflags.WithDefault("http://localhost"),
		testflags.WithFlagSet(fs),
	)
	client := axiom.DefineResource("client", func(r *axiom.Runner) (string, func(), error) {
		return endpoint.Get(r), nil, nil
	})
	runner := axiom.NewRunner(
		axiom.WithRunnerResources(testflags.Resource(testflags.WithSource(fs)), client),
	)
	if err := fs.Parse([]string{"-api.url=https://example.test"}); err != nil {
		panic(err)
	}
	fmt.Println(client.Get(runner))
	// Output: https://example.test
}
