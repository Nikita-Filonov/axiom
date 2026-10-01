package testflags_test

import (
	"flag"
	"os"
	"os/exec"
	"testing"

	"github.com/Nikita-Filonov/axiom"
	"github.com/Nikita-Filonov/axiom/plugins/testflags"
	"github.com/stretchr/testify/require"
)

// These declarations exercise registration before Go's test flag parser and
// creating a runner before parsing, exactly as a consuming package would.
var cliCustom = testflags.Int(testflags.WithName("axiom.custom"), testflags.WithDefault(0), testflags.WithUsage("custom value"))
var cliRunner = axiom.NewRunner(axiom.WithRunnerResources(testflags.Resource()))
var preParseError error
var beforeAllValue int

func TestMain(m *testing.M) {
	if os.Getenv("AXIOM_TESTFLAGS_HELPER") == "testmain" {
		_, preParseError = testflags.TryGet(cliRunner)
		flag.Parse()
		axiom.WithRunnerHooks(axiom.WithBeforeAll(func(r *axiom.Runner) {
			beforeAllValue = cliCustom.Get(r)
		}))(cliRunner)
		os.Exit(axiom.RunPackage(m, cliRunner))
	}
	os.Exit(m.Run())
}

func TestCLIChild(t *testing.T) {
	mode := os.Getenv("AXIOM_TESTFLAGS_HELPER")
	if mode == "" {
		t.Skip("subprocess only")
	}
	want := 55
	if mode == "default" {
		want = 0
	}
	require.Equal(t, want, cliCustom.Get(cliRunner))
	entry, _ := testflags.Get(cliRunner).Lookup("axiom.custom")
	require.Equal(t, mode != "default", entry.Set)
	_, ok := testflags.Get(cliRunner).Get("test.run")
	require.True(t, ok)
	if mode == "testmain" {
		require.ErrorContains(t, preParseError, "not been parsed")
		require.Equal(t, 55, beforeAllValue)
	}
}

func TestRealTestBinaryFlags(t *testing.T) {
	executable, err := os.Executable()
	require.NoError(t, err)
	for _, tc := range []struct {
		name string
		mode string
		args []string
		fail string
	}{
		{"default", "default", nil, ""},
		{"custom", "custom", []string{"-axiom.custom=55"}, ""},
		{"TestMain", "testmain", []string{"-axiom.custom=55"}, ""},
		{"repeated", "custom", []string{"-axiom.custom=1", "-axiom.custom=55"}, ""},
		{"invalid", "custom", []string{"-axiom.custom=wrong"}, "invalid value"},
		{"unknown", "custom", []string{"-axiom.unknown=55"}, "flag provided but not defined"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"-test.run=^TestCLIChild$"}, tc.args...)
			cmd := exec.Command(executable, args...)
			cmd.Env = append(os.Environ(), "AXIOM_TESTFLAGS_HELPER="+tc.mode)
			output, err := cmd.CombinedOutput()
			if tc.fail != "" {
				require.Error(t, err)
				require.Contains(t, string(output), tc.fail)
				return
			}
			require.NoError(t, err, "%s", output)
		})
	}
}
