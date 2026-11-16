package media

import (
	"image"
	"image/color"
	"math"
)

type span struct {
	start   int
	weights []float32
}

func buildSpans(srcLen, dstLen int) []span {
	spans := make([]span, dstLen)
	ratio := float64(srcLen) / float64(dstLen)
	for d := 0; d < dstLen; d++ {
		lo := float64(d) * ratio
		hi := lo + ratio
		first := int(math.Floor(lo))
		last := int(math.Ceil(hi))
		if last > srcLen {
			last = srcLen
		}
		weights := make([]float32, 0, last-first)
		for i := first; i < last; i++ {
			overlap := math.Min(float64(i+1), hi) - math.Max(float64(i), lo)
			weights = append(weights, float32(overlap/ratio))
		}
		spans[d] = span{start: first, weights: weights}
	}
	return spans
}

func fillRow(src image.Image, y int, row []uint8) {
	b := src.Bounds()
	w := b.Dx()
	yy := y + b.Min.Y
	switch img := src.(type) {
	case *image.YCbCr:
		for x := 0; x < w; x++ {
			xx := x + b.Min.X
			yi := img.YOffset(xx, yy)
			ci := img.COffset(xx, yy)
			r, g, bl := color.YCbCrToRGB(img.Y[yi], img.Cb[ci], img.Cr[ci])
			row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = r, g, bl, 255
		}
	case *image.RGBA:
		off := img.PixOffset(b.Min.X, yy)
		copy(row, img.Pix[off:off+w*4])
	case *image.NRGBA:
		off := img.PixOffset(b.Min.X, yy)
		for x := 0; x < w; x++ {
			p := img.Pix[off+x*4 : off+x*4+4]
			a := uint32(p[3])
			row[x*4] = uint8(uint32(p[0]) * a / 255)
			row[x*4+1] = uint8(uint32(p[1]) * a / 255)
			row[x*4+2] = uint8(uint32(p[2]) * a / 255)
			row[x*4+3] = p[3]
		}
	case *image.Gray:
		off := img.PixOffset(b.Min.X, yy)
		for x := 0; x < w; x++ {
			v := img.Pix[off+x]
			row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = v, v, v, 255
		}
	default:
		for x := 0; x < w; x++ {
			r, g, bl, a := src.At(x+b.Min.X, yy).RGBA()
			row[x*4], row[x*4+1], row[x*4+2], row[x*4+3] = uint8(r>>8), uint8(g>>8), uint8(bl>>8), uint8(a>>8)
		}
	}
}

func resample(src image.Image, dw, dh int) *image.RGBA {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	cols := buildSpans(sw, dw)
	row := make([]uint8, sw*4)
	reduced := make([]float32, dw*4)
	acc := make([]float32, dw*4)
	lastRow := -1
	vRatio := float64(sh) / float64(dh)
	for dy := 0; dy < dh; dy++ {
		for i := range acc {
			acc[i] = 0
		}
		lo := float64(dy) * vRatio
		hi := lo + vRatio
		last := int(math.Ceil(hi))
		if last > sh {
			last = sh
		}
		for y := int(math.Floor(lo)); y < last; y++ {
			weight := float32((math.Min(float64(y+1), hi) - math.Max(float64(y), lo)) / vRatio)
			if y != lastRow {
				fillRow(src, y, row)
				for dx, sp := range cols {
					var r, g, bl, a float32
					for k, wt := range sp.weights {
						p := (sp.start + k) * 4
						r += wt * float32(row[p])
						g += wt * float32(row[p+1])
						bl += wt * float32(row[p+2])
						a += wt * float32(row[p+3])
					}
					reduced[dx*4], reduced[dx*4+1], reduced[dx*4+2], reduced[dx*4+3] = r, g, bl, a
				}
				lastRow = y
			}
			for i, v := range reduced {
				acc[i] += v * weight
			}
		}
		out := dst.Pix[dy*dst.Stride : dy*dst.Stride+dw*4]
		for i, v := range acc {
			v += 0.5
			if v > 255 {
				v = 255
			}
			out[i] = uint8(v)
		}
	}
	return dst
}

func orient(src *image.RGBA, o int) *image.RGBA {
	if o < 2 || o > 8 {
		return src
	}
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	dw, dh := w, h
	if o >= 5 {
		dw, dh = h, w
	}
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var nx, ny int
			switch o {
			case 2:
				nx, ny = w-1-x, y
			case 3:
				nx, ny = w-1-x, h-1-y
			case 4:
				nx, ny = x, h-1-y
			case 5:
				nx, ny = y, x
			case 6:
				nx, ny = h-1-y, x
			case 7:
				nx, ny = h-1-y, w-1-x
			case 8:
				nx, ny = y, w-1-x
			}
			si := y*src.Stride + x*4
			di := ny*dst.Stride + nx*4
			copy(dst.Pix[di:di+4], src.Pix[si:si+4])
		}
	}
	return dst
}
