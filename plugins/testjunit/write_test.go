package testjunit

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type faultWriter struct {
	calls  int
	failAt int
	err    error
	short  bool
}

func (w *faultWriter) Write(p []byte) (int, error) {
	w.calls++
	if w.calls == w.failAt {
		if w.short {
			return len(p) - 1, nil
		}
		return 0, w.err
	}
	return len(p), nil
}

func TestWriteErrorsAtEveryStage(t *testing.T) {
	want := errors.New("output unavailable")
	for stage := 1; stage <= 3; stage++ {
		for _, short := range []bool{false, true} {
			w := &faultWriter{failAt: stage, err: want, short: short}
			err := NewReporter().Write(w)
			expected := want
			if short {
				expected = io.ErrShortWrite
			}
			require.ErrorIs(t, err, expected, "stage=%d, short=%v", stage, short)
			assert.Equal(t, stage, w.calls, "stage=%d, short=%v", stage, short)
		}
	}
	var nilBuffer *bytes.Buffer
	require.Error(t, NewReporter().Write(nilBuffer), "typed nil writer accepted")
}
