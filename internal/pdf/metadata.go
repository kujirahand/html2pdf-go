package pdf

import (
	"fmt"
	"strings"
	"unicode/utf16"
)

// Info contains the document information dictionary values.
type Info struct{ Title, Author, Subject, Keywords, Creator, Producer string }

// SetInfo sets document information.
func (w *Writer) SetInfo(info Info) { w.info = info }

func pdfString(s string) string {
	ascii := true
	for _, r := range s {
		if r < 0x20 || r > 0x7e {
			ascii = false
			break
		}
	}
	if ascii {
		var b strings.Builder
		b.WriteByte('(')
		for _, r := range s {
			if r == '(' || r == ')' || r == '\\' {
				b.WriteByte('\\')
			}
			b.WriteRune(r)
		}
		b.WriteByte(')')
		return b.String()
	}
	var b strings.Builder
	b.WriteString("<FEFF")
	for _, u := range utf16.Encode([]rune(s)) {
		fmt.Fprintf(&b, "%04X", u)
	}
	b.WriteByte('>')
	return b.String()
}

func (w *Writer) writeInfo(id int) {
	var b strings.Builder
	b.WriteString("<<")
	for _, v := range []struct{ name, value string }{{"Title", w.info.Title}, {"Author", w.info.Author}, {"Subject", w.info.Subject}, {"Keywords", w.info.Keywords}, {"Creator", w.info.Creator}, {"Producer", w.info.Producer}} {
		if v.value != "" {
			fmt.Fprintf(&b, " /%s %s", v.name, pdfString(v.value))
		}
	}
	b.WriteString(" >>")
	w.obj(id, b.String())
}
