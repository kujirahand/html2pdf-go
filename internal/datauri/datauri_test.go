package datauri

import (
	"bytes"
	"encoding/base64"
	"strings"
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

func TestDecodeParameterSyntax(t *testing.T) {
	enc := base64.StdEncoding.EncodeToString([]byte("hello"))
	for _, src := range []string{
		"data:image/png;base64;charset=utf-8," + enc,
		"data:image/png;base64;base64," + enc,
		"data:image/png;garbage,abc",
	} {
		if _, ok, err := Decode(src); !ok || err == nil {
			t.Errorf("Decode(%q): ok=%v err=%v, want error", src, ok, err)
		}
	}
	got, ok, err := Decode("data:image/png;charset=utf-8;base64," + enc)
	if !ok || err != nil || string(got) != "hello" {
		t.Errorf("got %q, %v, %v", got, ok, err)
	}
}

func TestDecodeTooLarge(t *testing.T) {
	_, ok, err := Decode("data:image/png," + strings.Repeat("A", MaxBytes+1))
	if !ok || err == nil || !strings.Contains(err.Error(), "too large") {
		t.Errorf("percent: ok=%v err=%v, want too large", ok, err)
	}
	big := strings.Repeat("QUJD", MaxBytes/3+1)
	_, ok, err = Decode("data:image/png;base64," + big)
	if !ok || err == nil || !strings.Contains(err.Error(), "too large") {
		t.Errorf("base64: ok=%v err=%v, want too large", ok, err)
	}
}

func TestDecodeParameterTokenSyntax(t *testing.T) {
	enc := base64.StdEncoding.EncodeToString([]byte("hello"))
	for _, src := range []string{
		"data:image/png;bad name=x;base64," + enc, // space in attribute name
		"data:image/png;foo=;base64," + enc,       // empty unquoted value
		"data:image/png;=x;base64," + enc,         // empty attribute name
	} {
		if _, _, err := Decode(src); err == nil {
			t.Errorf("Decode(%q) succeeded, want invalid parameter error", src)
		}
	}
	// A quoted-string value is valid.
	if _, _, err := Decode("data:image/png;charset=\"utf-8\";base64," + enc); err != nil {
		t.Errorf("quoted-string parameter rejected: %v", err)
	}
}

func TestDecodeTooLargeCheckedBeforeDecoding(t *testing.T) {
	// Every escape here is invalid, so decoding would fail with an escape
	// error; getting the too-large error instead proves the size is
	// checked from the scanned length before any decoding happens.
	src := "data:image/png," + strings.Repeat("%zz", MaxBytes)
	if _, _, err := Decode(src); err == nil || !strings.Contains(err.Error(), "too large") {
		t.Errorf("err = %v, want a too-large error", err)
	}
}

func TestDecodeNotDataURI(t *testing.T) {
	for _, src := range []string{"", "a.png", "http://x/y.png", "dat:x,y"} {
		if d, ok, err := Decode(src); ok || err != nil || d != nil {
			t.Errorf("Decode(%q) = %v, %v, %v", src, d, ok, err)
		}
	}
}
