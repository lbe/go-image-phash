package phash

import (
	"image"

	"github.com/lbe/go-image-phash/dct"
)

// HashFromLuminance runs a row-major luminance block through the exported dct
// package and then the full bit pipeline, turning it and the given Options
// into a Hash, the Go analogue of Image::PHash 0.3's pHash method applied to
// raw pixels: dctLuminance computes the 2D DCT of the block, and HashFromDCT
// applies the optional mirror or mirrorproof transform, the geometry / reduce
// / method coefficient selection with its extras and skip flag, the
// threshold conversion to bits, and the hex conversion of the resulting Hash.
//
// The caller's luminance slice is never mutated: dctLuminance hands the
// exported DCT a copy, matching the pipeline's contract of leaving caller
// data untouched.
func HashFromLuminance(luminance []float64, resize int, opts Options) (Hash, error) {
	return HashFromDCT(dctLuminance(luminance, resize), resize, opts)
}

// Luminance converts a resized n×n RGBA image into a row-major slice of n*n
// grayscale floats, the Go analogue of Image::PHash 0.3's Imager pixel step:
// each pixel becomes lumaOf's r*0.3 + g*0.59 + b*0.11 on its 0–255 channels,
// in scan order (y outer, x inner) exactly as _pixels_Imager pushes $grey.
// The returned slice owns its storage, so the caller may release any pooled
// RGBA buffer the image was scaled into before hashing.
//
// The walk reads img.Pix straight along Stride with integer offsets instead
// of going through per-pixel accessors, so the loop allocates only the
// returned slice.
func Luminance(img *image.RGBA, n int) []float64 {
	grey := make([]float64, n*n)
	pix := img.Pix
	i := 0
	for y := 0; y < n; y++ {
		row := y * img.Stride
		for x := 0; x < n; x++ {
			p := row + x*4
			grey[i] = lumaOf(pix[p], pix[p+1], pix[p+2])
			i++
		}
	}
	return grey
}

// lumaOf converts one pixel's 0–255 RGB channels into the grayscale float
// Image::PHash 0.3's pixel step computes: r*0.3 + g*0.59 + b*0.11, the
// backend weights (not BT.601's 0.299/0.587/0.114), matching the $grey
// _pixels_Imager pushes. The alpha channel never enters the formula.
func lumaOf(r, g, b uint8) float64 {
	return float64(r)*0.3 + float64(g)*0.59 + float64(b)*0.11
}

// dctLuminance computes the 2D DCT of a row-major luminance block, returning
// a fresh coefficient vector and leaving the input untouched. The exported
// DCT_2D needs a pre-allocated result vector for its power-of-two fast paths
// and works in place for other sizes, so the block is copied first and the
// caller's slice never changes.
func dctLuminance(luminance []float64, resize int) []float64 {
	coeffs := make([]float64, len(luminance))
	input := copyDCT(luminance)
	dct.DCT_2D(&input, resize, &coeffs)
	return coeffs
}
