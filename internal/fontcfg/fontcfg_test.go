package fontcfg

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/kujirahand/html2pdf-go/internal/text"
)

func TestGatherOrder(t *testing.T) {
	t.Setenv(EnvVar, filepath.Join("/env", "a")+string(filepath.ListSeparator)+filepath.Join("/env", "b"))
	got := Gather("/flag/a, /flag/b")
	want := []Dir{
		{"/flag/a", "flag"},
		{"/flag/b", "flag"},
		{filepath.Join("/env", "a"), "env"},
		{filepath.Join("/env", "b"), "env"},
	}
	for _, p := range text.DefaultFontDirs() {
		want = append(want, Dir{p, "os"})
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Gather = %v, want %v", got, want)
	}
}

func TestGatherDedupAndEmpty(t *testing.T) {
	os0 := text.DefaultFontDirs()[0]
	t.Setenv(EnvVar, "/x"+string(filepath.ListSeparator)+string(filepath.ListSeparator)+os0)
	got := Gather(",/x,,/y,/x")
	want := []Dir{{"/x", "flag"}, {"/y", "flag"}, {os0, "env"}}
	for _, p := range text.DefaultFontDirs()[1:] {
		want = append(want, Dir{p, "os"})
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Gather = %v, want %v", got, want)
	}
}

func TestGatherDedupNormalizesPaths(t *testing.T) {
	t.Setenv(EnvVar, "/x/")
	got := Gather("/x/,./y,y/")
	want := []Dir{{"/x", "flag"}, {"y", "flag"}}
	for _, p := range text.DefaultFontDirs() {
		want = append(want, Dir{p, "os"})
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Gather = %v, want %v", got, want)
	}
}

func TestGatherEnvOnlyAndNone(t *testing.T) {
	t.Setenv(EnvVar, "")
	got := Gather("")
	if !reflect.DeepEqual(Paths(got), text.DefaultFontDirs()) {
		t.Errorf("Paths = %v, want OS defaults", Paths(got))
	}
	for _, d := range got {
		if d.Origin != "os" {
			t.Errorf("origin of %q = %q, want os", d.Path, d.Origin)
		}
	}
}
