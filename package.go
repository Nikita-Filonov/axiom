package axiom

import "testing"

// RunPackage runs m within r's lifecycle. Call it from TestMain when several
// top level tests share a Runner. BeforeAll runs before m.Run; AfterAll and
// resource cleanups run after it, including if it panics. The exit code from
// m.Run is returned unchanged. If BeforeAll panics, m.Run and AfterAll do not
// run. RunPackage panics if m or r is nil.
func RunPackage(m *testing.M, r *Runner) int {
	if m == nil {
		panic("runpackage: nil *testing.M")
	}
	return RunPackageWith(r, m.Run)
}

// RunPackageWith runs entry within r's lifecycle and returns its exit code.
// It is the custom entry point counterpart of [RunPackage]. It panics if r or
// entry is nil.
func RunPackageWith(r *Runner, entry func() int) int {
	if r == nil {
		panic("runpackage: nil *Runner")
	}
	if entry == nil {
		panic("runpackage: nil entry function")
	}

	r.managed.Store(true)
	defer r.managed.Store(false)

	r.ApplyStart()
	defer r.ApplyFinish()
	return entry()
}
