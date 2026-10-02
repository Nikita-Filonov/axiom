package testjunit

import (
	"encoding/xml"
	"errors"
	"io"
	"reflect"
	"strings"
)

// Write writes a complete JUnit XML document for attempts finished so far.
// Export after the parent test or m.Run returns to include late failures.
// A nil Reporter or Writer (including a nil pointer) returns an error.
// Concurrent calls must use separate writers or a writer with synchronization.
func (r *Reporter) Write(w io.Writer) error {
	if r == nil {
		return errors.New("testjunit: nil reporter")
	}
	if w == nil || (reflect.ValueOf(w).Kind() == reflect.Pointer && reflect.ValueOf(w).IsNil()) {
		return errors.New("testjunit: nil writer")
	}
	if _, err := io.Copy(w, strings.NewReader(xml.Header)); err != nil {
		return err
	}
	encoder := xml.NewEncoder(w)
	encoder.Indent("", "  ")
	if err := encoder.Encode(r.snapshot().document()); err != nil {
		return err
	}
	_, err := io.Copy(w, strings.NewReader("\n"))
	return err
}
