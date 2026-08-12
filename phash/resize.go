package phash

import (
	"image"

	"golang.org/x/image/draw"

	"github.com/lbe/go-image-phash/internal/gensyncpool"
)

// Resizer scales a source image into a standalone *image.RGBA destination of
// the requested size. A nil Resizer selects the package's default scaler,
// which draws with golang.org/x/image/draw's ApproxBiLinear into a buffer
// checked out of the pool.
type Resizer interface {
	// Resize scales src to an n×n RGBA and returns it. The destination is
	// owned by the caller.
	Resize(src image.Image, n int) *image.RGBA
}

// resizePool pools the N×N RGBA canvases the default scaler draws into. The
// init allocates canvases at the default resize size; a canvas checked out
// for a smaller N is re-sliced in place, and a larger request gets a fresh
// allocation. Canvases are zeroed when returned so a reused canvas never
// carries stale pixels.
//
// Lifetime: a checked-out canvas belongs to its borrower until the release
// func from Resize/scaleWith runs, and the borrower must not retain the
// canvas or its Pix after that. The hash pipeline observes this by copying
// the luminance into owned storage before releasing the checkout.
var resizePool = gensyncpool.New(
	func() *image.RGBA { return image.NewRGBA(image.Rect(0, 0, defaultResize, defaultResize)) },
	func(img *image.RGBA) { clear(img.Pix) },
)

// scaleWith scales src to an n×n canvas drawn from resizePool using scaler.
// The destination stays checked out of the pool — and its backing store must
// not be retained after release — until the returned release func runs, which
// returns the buffer to the pool.
func scaleWith(src image.Image, n int, scaler draw.Scaler) (*image.RGBA, func()) {
	dst := resizePool.Get()
	if need := n * n * 4; cap(dst.Pix) < need {
		// The pooled canvas is too small for this request; use a fresh one.
		dst = image.NewRGBA(image.Rect(0, 0, n, n))
	} else {
		dst.Pix = dst.Pix[:need]
		dst.Rect = image.Rect(0, 0, n, n)
		dst.Stride = n * 4
	}
	scaler.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst, func() { resizePool.Put(dst) }
}

// Resize scales src to an n×n *image.RGBA using r, or the default
// ApproxBiLinear scaler when r is nil. The returned destination stays
// checked out of the pool — and its backing store must not be retained after
// release — until the returned release func runs, which returns the buffer to
// the pool (a no-op for destinations produced by a supplied Resizer).
func Resize(src image.Image, n int, r Resizer) (dst *image.RGBA, release func()) {
	if r != nil {
		return r.Resize(src, n), func() {}
	}
	return scaleWith(src, n, draw.ApproxBiLinear)
}
