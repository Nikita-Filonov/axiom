// Package testleaks checks for goroutines created by a case attempt that remain
// alive after the body and its later-registered cleanups, and for explicitly
// tracked resources not released by then. Goroutine attribution uses pprof
// labels inherited by child goroutines.
package testleaks
