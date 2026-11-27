package datauri

import (
	"bytes"
	"encoding/base64"
	"testing"
)

func TestDecodeBase64(t *testing.T) {
	want := []byte("\x89PNG\r\n\x1a\nhello")
	enc := base64.StdEncoding.EncodeToString(want)
	for _, src := range []string{
		"data:image/png;base64," + enc,
		"DATA:IMAGE/PNG;BASE64," + enc,
		"data:;base64," + enc,
		"data:image/png;base64," + enc[:4] + " \n" + enc[4:],
		"data:image/png;base64," + base64.RawStdEncoding.EncodeToString(want),
	} {
		got, ok, err := Decode(src)
		if !ok || err != nil || !bytes.Equal(got, want) {
			t.Errorf("Decode(%q) = %q, %v, %v", src, got, ok, err)
		}
	}
}

func TestDecodePercent(t *testing.T) {
	got, ok, err := Decode("data:image/gif,a%20b+c")
	if !ok || err != nil || string(got) != "a b+c" {
		t.Errorf("got %q, %v, %v", got, ok, err)
	}
}

func TestDecodeErrors(t *testing.T) {
	for _, src := range []string{
		"data:text/plain;base64,aGVsbG8=",
		"data:image/png;base64",
		"data:image/png;base64,@@@@!",
		"data:image/png,%zz",
	} {
		if _, ok, err := Decode(src); !ok || err == nil {
			t.Errorf("Decode(%q): ok=%v err=%v, want error", src, ok, err)
		}
	}
}

func TestDecodeNotDataURI(t *testing.T) {
	for _, src := range []string{"", "a.png", "http://x/y.png", "dat:x,y"} {
		if d, ok, err := Decode(src); ok || err != nil || d != nil {
			t.Errorf("Decode(%q) = %v, %v, %v", src, d, ok, err)
		}
	}
}
