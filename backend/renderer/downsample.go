package renderer

import (
	"image"
	"image/color"
)

func downsample(img *image.NRGBA, scaleFactor int) *image.NRGBA {
	if scaleFactor <= 1 {
		return img
	}
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	ow := w / scaleFactor
	oh := h / scaleFactor
	out := image.NewNRGBA(image.Rect(0, 0, ow, oh))
	for y := 0; y < oh; y++ {
		for x := 0; x < ow; x++ {
			var r, g, b, a uint32
			for dy := 0; dy < scaleFactor; dy++ {
				for dx := 0; dx < scaleFactor; dx++ {
					c := img.NRGBAAt(x*scaleFactor+dx, y*scaleFactor+dy)
					cr, cg, cb, ca := c.RGBA()
					r += cr
					g += cg
					b += cb
					a += ca
				}
			}
			n := uint32(scaleFactor * scaleFactor)
			out.SetNRGBA(x, y, color.NRGBA{
				R: uint8((r / n) >> 8),
				G: uint8((g / n) >> 8),
				B: uint8((b / n) >> 8),
				A: uint8((a / n) >> 8),
			})
		}
	}
	return out
}