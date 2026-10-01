// Package testflags exposes command-line flags through typed keys and a lazy
// runner resource. Declare flags before Go parses arguments, usually in package
// variables, and register Resource with axiom.WithRunnerResources on each runner
// that needs them.
//
// Go parses arguments before ordinary tests. In TestMain, call flag.Parse before
// reading flags or running BeforeAll hooks that read them. This package never
// parses arguments, changes os.Args, or registers flags merely by being imported.
// Finish registration, parsing, and any flag.Set calls before concurrent reads.
package testflags
