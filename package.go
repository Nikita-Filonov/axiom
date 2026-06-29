package axiom

import "testing"

func RunPackage(m *testing.M, r *Runner) int {
	if m == nil {
		panic("runpackage: nil *testing.M")
	}
	return RunPackageWith(r, m.Run)
}

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
