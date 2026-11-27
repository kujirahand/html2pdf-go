package layout

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"math"
	"testing"
)

func testPNGDataURI(t *testing.T, w, h int) string {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, w, h))); err != nil {
		t.Fatal(err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

func imageItems(pages []*Page) []ImageItem {
	var out []ImageItem
	for _, p := range pages {
		for _, it := range p.Items {
			if im, ok := it.(ImageItem); ok {
				out = append(out, im)
			}
		}
	}
	return out
}

func TestDataURIImageSize(t *testing.T) {
	uri := testPNGDataURI(t, 40, 20)
	items := imageItems(run(t, `<body><p>x <img src="`+uri+`"></p></body>`))
	if len(items) != 1 {
		t.Fatalf("got %d image items, want 1", len(items))
	}
	if items[0].Src != uri {
		t.Error("ImageItem.Src should carry the data URI")
	}
	if math.Abs(items[0].W-30) > 0.01 || math.Abs(items[0].H-15) > 0.01 {
		t.Errorf("size = %vx%v, want 30x15", items[0].W, items[0].H)
	}

	items = imageItems(run(t, `<body><p>x <img src="`+uri+`" style="width:100px"></p></body>`))
	if len(items) != 1 {
		t.Fatalf("got %d image items, want 1", len(items))
	}
	if math.Abs(items[0].W-75) > 0.01 || math.Abs(items[0].H-37.5) > 0.01 {
		t.Errorf("size = %vx%v, want 75x37.5", items[0].W, items[0].H)
	}
}

func TestDataURIImageFormats(t *testing.T) {
	var g, j bytes.Buffer
	if err := gif.Encode(&g, image.NewRGBA(image.Rect(0, 0, 40, 20)), nil); err != nil {
		t.Fatal(err)
	}
	if err := jpeg.Encode(&j, image.NewRGBA(image.Rect(0, 0, 40, 20)), nil); err != nil {
		t.Fatal(err)
	}
	for _, uri := range []string{
		"data:image/gif;base64," + base64.StdEncoding.EncodeToString(g.Bytes()),
		"data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(j.Bytes()),
	} {
		items := imageItems(run(t, `<body><p>x <img src="`+uri+`"></p></body>`))
		if len(items) != 1 {
			t.Fatalf("%.20s: got %d image items, want 1", uri, len(items))
		}
		if math.Abs(items[0].W-30) > 0.01 || math.Abs(items[0].H-15) > 0.01 {
			t.Errorf("%.20s: size = %vx%v, want 30x15", uri, items[0].W, items[0].H)
		}
	}
}

func TestRemoteImageSkipped(t *testing.T) {
	for _, src := range []string{"http://example.com/x.png", "https://example.com/x.png"} {
		if items := imageItems(run(t, `<body><p>x <img src="`+src+`"></p></body>`)); len(items) != 0 {
			t.Errorf("src %q: got %d image items, want 0", src, len(items))
		}
	}
}

func TestDataURIImageInvalid(t *testing.T) {
	for _, src := range []string{
		"data:image/png;base64,!!!!",
		"data:image/png;base64,aGVsbG8=",
		"data:text/plain;base64,aGVsbG8=",
		"data:image/png",
	} {
		if items := imageItems(run(t, `<body><p>x <img src="`+src+`"></p></body>`)); len(items) != 0 {
			t.Errorf("src %q: got %d image items, want 0", src, len(items))
		}
	}
}
