package phash

import (
	"errors"
	"image"
	"os"
	"sync"
)

// PHash is the Go analogue of Image::PHash 0.3's image object: it holds the
// source image and the resize setting and computes a perceptual hash from
// them on demand. The first hash caches the resize and DCT results as owned
// memory: the pooled resize buffer is released as soon as the luminance is
// copied out of it, and the DCT coefficient vector is stored on the instance,
// so later option variants hash the same cached DCT instead of recomputing
// from the source, and no result ever depends on a later resize reusing the
// pool.
type PHash struct {
	img    image.Image
	resize int

	// dctOnce guards the lazy computation of the DCT coefficient vector,
	// exposed through dct(). The resize, luminance, and DCT steps depend
	// only on the constructor's image and resize, not on per-call options,
	// so they run once and every later PHash or DCTDump call reuses the
	// owned coefficients.
	dctOnce sync.Once
	coeffs  []float64

	// reducedOnce guards the lazy on-demand resize shared by ReducedImage
	// and computeDCT. The scaled image depends only on the constructor's
	// image and resize, so it runs once and every later call — ReducedImage
	// or the DCT path — reuses the same cached image. The cache is owned
	// storage, never a pooled buffer.
	reducedOnce sync.Once
	reduced     *image.RGBA
}

// NewFromPath loads the image file at path and constructs a PHash from it
// with the given resize and options, the Go analogue of Image::PHash 0.3's
// image-object load: a missing or undecodable file is a load error, not a
// silent nil, and the resize value is validated like NewFromImage's. Decode
// formats are registered by the package (see decode.go), so NewFromPath can
// decode them without the caller importing anything.
func NewFromPath(path string, resize int, opts Options) (_ *PHash, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	return NewFromImage(img, resize, opts)
}

// NewFromImage constructs a PHash from an in-memory image, the Go analogue of
// Image::PHash 0.3's image-object constructor: validateResize normalizes and
// rejects the resize value before the instance is returned. The options
// argument mirrors the Perl constructor's settings parameter, but hashing
// options are taken per call by PHash, so they are not retained on the
// instance.
func NewFromImage(img image.Image, resize int, opts Options) (*PHash, error) {
	resize, err := validateResize(resize)
	if err != nil {
		return nil, err
	}
	return &PHash{img: img, resize: resize}, nil
}

// validateResize normalizes a constructor resize: a non-positive value
// defaults to 32 (Image::PHash's default DCT size), and any remaining value
// must be > 5, matching the Perl constructor's "resize > 5 expected." croak.
// It returns the effective resize, or an error when the value is rejected.
func validateResize(resize int) (int, error) {
	if resize <= 0 {
		resize = defaultResize
	}
	if resize <= 5 {
		return 0, errors.New("resize > 5 expected")
	}
	return resize, nil
}

// PHash computes the perceptual hash of the instance's image for the given
// options, the Go analogue of Image::PHash 0.3's pHash method: the first call
// triggers the on-demand resize through ReducedImage — the scaled pixels are
// copied into owned storage before the pooled buffer is released — runs the
// luminance and DCT steps over that owned image, and caches the resulting
// DCT on the instance; every call then applies option validation (including
// the mutually exclusive mirror/mirrorproof pair) and the bit pipeline to
// that cached coefficient vector, so a later call with different options
// still hashes the content of the first call, and the source image may be
// changed or the pool reused without affecting it.
//
// Pool lifetime: the canvas Resize checks out of resizePool belongs to the
// pool until release runs, so nothing after the checkout may reference it.
// ReducedImage copies the scaled pixels into owned storage before release,
// computeDCT reads the luminance from that owned image, and dctLuminance
// copies the block again into owned coefficient storage, which is why the
// release may run before hashing: a later instance resizing a different image
// into a reused buffer never corrupts this instance's cached results.
// HashFromDCT leaves the cached vector untouched (mirror/mirrorproof return
// fresh slices and the selection and threshold steps only read), so the cache
// survives across option variants.
func (p *PHash) PHash(opts Options) (Hash, error) {
	return HashFromDCT(p.dct(), p.resize, opts)
}

// PHash7 computes the 7x7 reduced perceptual hash of the instance's image,
// the Go analogue of Image::PHash 0.3's pHash7 method: it delegates to
// PHash with the geometry '7x7' and reduce settings, i.e.
// PHash(Options{Geometry: "7x7", Reduce: true}).
func (p *PHash) PHash7() (Hash, error) {
	return p.PHash(Options{Geometry: "7x7", Reduce: true})
}

// PHash6 computes the 6x6 reduced median perceptual hash of the instance's
// image, the Go analogue of Image::PHash 0.3's pHash6 method: it delegates to
// PHash with geometry '6x6', reduce, and method 'median', i.e.
// PHash(Options{Geometry: "6x6", Reduce: true, Method: "median"}).
func (p *PHash) PHash6() (Hash, error) {
	return p.PHash(Options{Geometry: "6x6", Reduce: true, Method: "median"})
}

// ReducedImage returns the scaled N×N image the DCT step hashes, the Go
// analogue of Image::PHash 0.3's reducedimage method: it triggers the resize
// on demand — the first call scales the source to the configured resize and
// caches the result, so later calls reuse the same scaled image without
// resizing again — and the returned image never references a pool buffer
// that has been Put: the scaled pixels are copied into owned storage before
// the pooled buffer is released, so a later instance resizing a different
// image into a reused pool buffer never corrupts this instance's image.
func (p *PHash) ReducedImage() image.Image {
	return p.scaled()
}

// DCTDump returns the full resize×resize DCT coefficient grid of the
// instance's image as a row-major slice of floats, the Go analogue of
// Image::PHash 0.3's dctdump method: it returns the same DCT vector the hash
// pipeline hashes — the DCT of the instance's own luminance block, computed
// and cached exactly like PHash's, so a dump never recomputes a second
// resize — and when opts requests the mirror or mirrorproof option the
// returned grid is that transform of the base DCT: _dct_mirror negates every
// other coefficient, _dct_mirrorproof takes its absolute value, and neither
// variant is ever the plain base dump; the mutually exclusive pair is
// rejected like Perl's _mirroring croak.
func (p *PHash) DCTDump(opts Options) ([]float64, error) {
	if err := checkMirrorExclusive(opts); err != nil {
		return nil, err
	}
	coeffs := p.dct()
	switch {
	case opts.Mirror:
		return mirrorDCT(coeffs), nil
	case opts.MirrorProof:
		return mirrorproofDCT(coeffs), nil
	}
	return coeffs, nil
}

// dct returns the owned DCT coefficient vector, computing it once on first
// use. PHash and DCTDump both go through this one cache, so a dump never
// recomputes a second resize and every later option variant hashes the same
// vector the first call computed. The coefficients are owned storage:
// dctLuminance copies the luminance block, so nothing references a pooled
// buffer after release.
func (p *PHash) dct() []float64 {
	p.dctOnce.Do(func() {
		p.coeffs = p.computeDCT()
	})
	return p.coeffs
}

// scaled returns the owned N×N scaled image the DCT path hashes, triggering
// the on-demand resize on first use. ReducedImage and computeDCT both go
// through this one cache, so the resize runs at most once per instance no
// matter which of them is called first. The cached image is owned storage,
// never a pooled buffer: the scaled pixels are copied out of the pooled
// destination before release.
func (p *PHash) scaled() *image.RGBA {
	p.reducedOnce.Do(func() {
		dst, release := Resize(p.img, p.resize, nil)
		owned := image.NewRGBA(image.Rect(0, 0, p.resize, p.resize))
		copy(owned.Pix, dst.Pix)
		release()
		p.reduced = owned
	})
	return p.reduced
}

// computeDCT runs the luminance→DCT steps that depend only on the
// constructor's image and resize, not on per-call options, and returns the
// owned coefficient vector dct() caches for every later option variant. It
// reuses the scaled image ReducedImage exposes on demand instead of checking
// out its own pooled buffer, so the DCT path shares the single per-instance
// resize with ReducedImage; dctLuminance copies the block into owned
// coefficient storage, leaving nothing referencing the pooled buffer after
// release.
func (p *PHash) computeDCT() []float64 {
	luminance := Luminance(p.scaled(), p.resize)
	return dctLuminance(luminance, p.resize)
}
