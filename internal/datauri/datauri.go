// Package datauri decodes RFC 2397 data: URIs for use as image sources.
package datauri

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// MaxBytes caps the decoded payload size to bound memory use.
const MaxBytes = 64 << 20

// Decode returns the payload of a data: URI. If src is not a data URI it
// returns isDataURI=false and no error. A data URI whose media type is
// present but not image/* is an error.
func Decode(src string) (data []byte, isDataURI bool, err error) {
	if len(src) < 5 || !strings.EqualFold(src[:5], "data:") {
		return nil, false, nil
	}
	rest := src[5:]
	comma := strings.IndexByte(rest, ',')
	if comma < 0 {
		return nil, true, errors.New("datauri: missing comma")
	}
	meta, payload := rest[:comma], rest[comma+1:]
	parts := strings.Split(meta, ";")
	if mt := strings.TrimSpace(parts[0]); mt != "" && !strings.HasPrefix(strings.ToLower(mt), "image/") {
		return nil, true, fmt.Errorf("datauri: media type %q is not an image", mt)
	}
	isBase64 := false
	for _, p := range parts[1:] {
		if strings.EqualFold(strings.TrimSpace(p), "base64") {
			isBase64 = true
		}
	}
	if isBase64 {
		data, err = decodeBase64(payload)
	} else {
		var s string
		s, err = url.PathUnescape(payload)
		data = []byte(s)
	}
	if err != nil {
		return nil, true, fmt.Errorf("datauri: invalid payload: %w", err)
	}
	if len(data) > MaxBytes {
		return nil, true, errors.New("datauri: payload too large")
	}
	return data, true, nil
}

func decodeBase64(s string) ([]byte, error) {
	if len(s) > MaxBytes/3*4+8 {
		return nil, errors.New("payload too large")
	}
	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			return -1
		}
		return r
	}, s)
	var firstErr error
	for _, enc := range []*base64.Encoding{base64.StdEncoding, base64.RawStdEncoding, base64.URLEncoding, base64.RawURLEncoding} {
		b, err := enc.DecodeString(s)
		if err == nil {
			return b, nil
		}
		if firstErr == nil {
			firstErr = err
		}
	}
	return nil, firstErr
}
