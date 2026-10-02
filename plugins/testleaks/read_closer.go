package testleaks

import (
	"io"
	"reflect"

	"github.com/Nikita-Filonov/axiom"
)

type trackedReadCloser struct {
	io.ReadCloser
	handle *Handle
}

func (c *trackedReadCloser) Close() error {
	err := c.ReadCloser.Close()
	c.handle.Release()
	return err
}

// TrackReadCloser wraps a ReadCloser and records whether Close was called.
// It tracks the call to Close, not whether Close returned a nil error.
func TrackReadCloser(cfg *axiom.Config, name string, value io.ReadCloser) io.ReadCloser {
	if value == nil || (reflect.ValueOf(value).Kind() == reflect.Pointer && reflect.ValueOf(value).IsNil()) {
		panic("testleaks: nil read closer")
	}
	return &trackedReadCloser{ReadCloser: value, handle: track(cfg, name)}
}
