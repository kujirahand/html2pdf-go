package pdf

import (
	"bytes"
	"fmt"
	"image/color"
	"image/png"
)

// Image is a decoded raster image to be embedded as an XObject.
type Image struct {
	index         int
	width, height int
	rgb           []byte
	alpha         []byte // nil if fully opaque
	objID         int
	smaskID       int
}

func (im *Image) name() string { return fmt.Sprintf("Im%d", im.index) }

// AddPNGImage decodes a PNG and registers it as an RGB image (with an alpha SMask if needed).
func (w *Writer) AddPNGImage(data []byte) (*Image, error) {
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("pdf: decode PNG: %w", err)
	}
	b := img.Bounds()
	if b.Dx() == 0 || b.Dy() == 0 {
		return nil, fmt.Errorf("pdf: PNG has empty size %dx%d", b.Dx(), b.Dy())
	}
	rgb := make([]byte, 0, b.Dx()*b.Dy()*3)
	alpha := make([]byte, 0, b.Dx()*b.Dy())
	hasAlpha := false
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			rgb = append(rgb, c.R, c.G, c.B)
			alpha = append(alpha, c.A)
			if c.A != 255 {
				hasAlpha = true
			}
		}
	}
	if !hasAlpha {
		alpha = nil
	}
	im := &Image{index: len(w.images), width: b.Dx(), height: b.Dy(), rgb: rgb, alpha: alpha}
	w.images = append(w.images, im)
	return im, nil
}

func (w *Writer) writeImage(im *Image) {
	base := fmt.Sprintf("/Type /XObject /Subtype /Image /Width %d /Height %d /BitsPerComponent 8", im.width, im.height)
	dict := base + " /ColorSpace /DeviceRGB"
	if im.alpha != nil {
		dict += fmt.Sprintf(" /SMask %d 0 R", im.smaskID)
		w.stream(im.smaskID, base+" /ColorSpace /DeviceGray", im.alpha, true)
	}
	w.stream(im.objID, dict, im.rgb, true)
}
