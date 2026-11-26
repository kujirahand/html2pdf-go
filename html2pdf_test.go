package html2pdf

import (
	"bytes"
	"errors"
	"testing"
)

func TestConvertNotImplemented(t *testing.T) {
	var buf bytes.Buffer
	err := Convert([]byte("<p>hello</p>"), &buf, nil)
	if !errors.Is(err, ErrNotImplemented) {
		t.Fatalf("Convert error = %v, want ErrNotImplemented", err)
	}
}
