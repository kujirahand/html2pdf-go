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
	params := parts[1:]
	for i, p := range params {
		p = strings.TrimSpace(p)
		if strings.EqualFold(p, "base64") {
			if i != len(params)-1 {
				return nil, true, errors.New("datauri: base64 parameter must be last")
			}
			isBase64 = true
			continue
		}
		if !validParam(p) {
			return nil, true, fmt.Errorf("datauri: invalid parameter %q", p)
		}
	}
	if isBase64 {
		data, err = decodeBase64(payload)
	} else {
		if percentDecodedLen(payload) > MaxBytes {
			return nil, true, errors.New("datauri: payload too large")
		}
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

// validParam reports whether p is a MIME parameter in attribute=value form
// (RFC 2045): the attribute is a token, and the value is a token or a
// quoted-string.
func validParam(p string) bool {
	k, v, ok := strings.Cut(p, "=")
	if !ok || !isToken(strings.TrimSpace(k)) {
		return false
	}
	v = strings.TrimSpace(v)
	if len(v) >= 2 && v[0] == '"' && v[len(v)-1] == '"' {
		return validQuoted(v[1 : len(v)-1])
	}
	return isToken(v)
}

// validQuoted reports whether s (the inside of a quoted-string) is valid
// qtext / quoted-pair content: printable ASCII and tab, excluding '"' and
// '\', where a backslash must be followed by a printable character or tab.
func validQuoted(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '\\' {
			i++
			if i >= len(s) || (s[i] != '\t' && (s[i] < ' ' || s[i] > '~')) {
				return false
			}
			continue
		}
		if c != '\t' && (c < ' ' || c > '~' || c == '"') {
			return false
		}
	}
	return true
}

// isToken reports whether s is an RFC 2045 token: one or more ASCII
// characters excluding controls, spaces and the tspecials separators.
func isToken(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c <= ' ' || c >= 0x7f || strings.IndexByte(`()<>@,;:\"/[]?=`, c) >= 0 {
			return false
		}
	}
	return true
}

// percentDecodedLen returns the decoded length of a percent-encoded string
// without allocating. Invalid escapes count as one byte; PathUnescape reports
// them later.
func percentDecodedLen(s string) int {
	n := 0
	for i := 0; i < len(s); n++ {
		if s[i] == '%' && i+2 < len(s) && isHex(s[i+1]) && isHex(s[i+2]) {
			i += 3
		} else {
			i++
		}
	}
	return n
}

func isHex(c byte) bool {
	return '0' <= c && c <= '9' || 'a' <= c && c <= 'f' || 'A' <= c && c <= 'F'
}

func decodeBase64(s string) ([]byte, error) {
	s = strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\n', '\r', '\f', '\v':
			return -1
		}
		return r
	}, s)
	if len(s) > MaxBytes/3*4+8 {
		return nil, errors.New("payload too large")
	}
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
