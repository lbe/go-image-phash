package phash_test

import (
	"image"
	"math"
	"reflect"
	"testing"

	"github.com/lbe/go-image-phash/dct"
	"github.com/lbe/go-image-phash/phash"
)

// dctDumpExpected computes the row-major DCT coefficient grid DCTDump must
// return for an instance whose on-demand scaled image is scaled: the exported
// 2D DCT of the instance's own luminance block, i.e. exactly the vector the
// hash pipeline DCTs. It is built only from independently verified pieces —
// ReducedImage (golden-tested), Luminance, and the exported dct package — so
// a DCTDump mismatch cannot be blamed on the expectation. The exported
// DCT_2D writes through its result pointer and transforms the input in place
// on its fast paths, so the luminance block is copied first and the caller's
// slice is left untouched.
func dctDumpExpected(t *testing.T, scaled *image.RGBA, n int) []float64 {
	t.Helper()
	luminance := phash.Luminance(scaled, n)
	input := append([]float64(nil), luminance...)
	coeffs := make([]float64, len(input))
	dct.DCT_2D(&input, n, &coeffs)
	return coeffs
}

// mirrorVariant transforms a row-major DCT coefficient grid the way
// Image::PHash 0.3's _dct_mirror does: every coefficient at an odd 0-based
// index is negated while the rest are kept, and the result is a fresh slice.
func mirrorVariant(coeffs []float64) []float64 {
	out := make([]float64, len(coeffs))
	for i, c := range coeffs {
		if i%2 == 1 {
			out[i] = -c
		} else {
			out[i] = c
		}
	}
	return out
}

// mirrorproofVariant transforms a row-major DCT coefficient grid the way
// Image::PHash 0.3's _dct_mirrorproof does: every coefficient at an odd
// 0-based index is replaced by its absolute value while the rest are kept,
// and the result is a fresh slice.
func mirrorproofVariant(coeffs []float64) []float64 {
	out := make([]float64, len(coeffs))
	for i, c := range coeffs {
		if i%2 == 1 {
			out[i] = math.Abs(c)
		} else {
			out[i] = c
		}
	}
	return out
}

// countNegativeOddCoefficients reports how many coefficients at an odd
// 0-based index are negative, the positions where the mirror and mirrorproof
// variants observably differ from the base DCT.
func countNegativeOddCoefficients(coeffs []float64) int {
	neg := 0
	for i := 1; i < len(coeffs); i += 2 {
		if coeffs[i] < 0 {
			neg++
		}
	}
	return neg
}

// previewDCT returns a compact preview of a float slice for error messages:
// the full slice when it is short, otherwise its first few elements.
func previewDCT(v []float64) []float64 {
	const preview = 6
	if len(v) <= preview {
		return v
	}
	return v[:preview]
}

// TestDCTDumpReturnsResizeSizedDCTOfInstanceLuminanceWithMirrorVariants
// verifies the DCTDump helper, the Go analogue of Image::PHash 0.3's dctdump
// method (which runs the reduce step unless im_scaled is set, computes and
// caches dct2d over the instance's pixels, and returns the base DCT or the
// _dct_mirror/_dct_mirrorproof variant the options request):
//
//   - the base dump (empty Options) is a slice of exactly resize×resize
//     floats whose values equal the exported 2D DCT of the instance's own
//     luminance block — for the checked-in 32×32 synthetic fixture with the
//     default resize 32 (identity scale) that is the DCT of the fixture's own
//     pixels, and for a configured resize of 8 it is the DCT of the 8×8
//     scaled image's luminance, so length and shape track the resize;
//   - Options{Mirror: true} returns the _dct_mirror variant (odd-index
//     coefficients negated) and Options{MirrorProof: true} the
//     _dct_mirrorproof variant (odd-index coefficients made absolute), each
//     compared to the corresponding transform of the base DCT and never
//     allowed to be the plain base dump — the fixture's DCT carries 170
//     negative odd-index coefficients, so the variants are observably
//     different from the base grid;
//   - Options{Mirror: true, MirrorProof: true} errors like Perl's _mirroring
//     croak ("Options 'mirror' and 'mirrorproof' are mutually exclusive"),
//     the same exclusivity Validate enforces for hashes.
func TestDCTDumpReturnsResizeSizedDCTOfInstanceLuminanceWithMirrorVariants(t *testing.T) {
	golden := loadSyntheticGolden(t)
	img := syntheticFixtureImage(t, golden)

	const n = 32 // the fixture's edge length and the default resize: identity scale
	p, err := phash.NewFromImage(img, 0, phash.Options{})
	if err != nil {
		t.Fatalf("NewFromImage(synthetic image, 0, Options{}) returned an unexpected error: %v", err)
	}
	base := dctDumpExpected(t, p.ReducedImage().(*image.RGBA), n)
	if len(base) != n*n {
		t.Fatalf("expected DCT has %d coefficients, want %d (N*N); the fixture or luminance is broken", len(base), n*n)
	}
	if countNegativeOddCoefficients(base) == 0 {
		t.Fatalf("fixture DCT has no negative odd-index coefficient; the mirror assertions could not tell the variants apart from the base dump")
	}

	t.Run("empty options return the full resize by resize DCT of the instance's luminance", func(t *testing.T) {
		got, err := p.DCTDump(phash.Options{})
		if err != nil {
			t.Fatalf("DCTDump(Options{}) returned an unexpected error: %v", err)
		}
		if got == nil {
			t.Fatalf("DCTDump(Options{}) returned a nil slice; want the %dx%d DCT grid (%d coefficients) of the instance's luminance", n, n, n*n)
		}
		if len(got) != n*n {
			t.Fatalf("DCTDump(Options{}) returned %d coefficients, want %d (resize %d squared: the full DCT grid, not a reduced selection)", len(got), n*n, n)
		}
		if !reflect.DeepEqual(got, base) {
			t.Errorf("DCTDump(Options{}) = %v..., want the %dx%d DCT of the instance's luminance %v... (Image::PHash dctdump returns dct2d over the instance's pixels)",
				previewDCT(got), n, n, previewDCT(base))
		}
	})

	t.Run("length and shape follow the configured resize", func(t *testing.T) {
		const small = 8
		p8, err := phash.NewFromImage(img, small, phash.Options{})
		if err != nil {
			t.Fatalf("NewFromImage(synthetic image, %d, Options{}) returned an unexpected error: %v", small, err)
		}
		got, err := p8.DCTDump(phash.Options{})
		if err != nil {
			t.Fatalf("DCTDump(Options{}) on a resize-%d instance returned an unexpected error: %v", small, err)
		}
		if len(got) != small*small {
			t.Fatalf("DCTDump(Options{}) on a resize-%d instance returned %d coefficients, want %d (the %dx%d grid: length and shape must track the configured resize)", small, len(got), small*small, small, small)
		}
		want := dctDumpExpected(t, p8.ReducedImage().(*image.RGBA), small)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("DCTDump(Options{}) on a resize-%d instance = %v..., want the %dx%d DCT of that instance's luminance %v...",
				small, previewDCT(got), small, small, previewDCT(want))
		}
	})

	t.Run("mirror option returns the _dct_mirror variant, not the base dump", func(t *testing.T) {
		got, err := p.DCTDump(phash.Options{Mirror: true})
		if err != nil {
			t.Fatalf("DCTDump(Options{Mirror: true}) returned an unexpected error: %v", err)
		}
		if len(got) != n*n {
			t.Fatalf("DCTDump(Options{Mirror: true}) returned %d coefficients, want %d (the full mirrored grid)", len(got), n*n)
		}
		want := mirrorVariant(base)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("DCTDump(Options{Mirror: true}) = %v..., want the _dct_mirror variant of the instance's DCT %v... (Image::PHash dctdump with the mirror option returns the _dct_mirror transform: every other coefficient negated)",
				previewDCT(got), previewDCT(want))
		}
		if reflect.DeepEqual(got, base) {
			t.Errorf("DCTDump(Options{Mirror: true}) returned the plain base dump; the mirror option must select the _dct_mirror variant, not the base DCT (the fixture has %d negative odd-index coefficients, so the variant is observable)",
				countNegativeOddCoefficients(base))
		}
	})

	t.Run("mirrorproof option returns the _dct_mirrorproof variant, not the base dump", func(t *testing.T) {
		got, err := p.DCTDump(phash.Options{MirrorProof: true})
		if err != nil {
			t.Fatalf("DCTDump(Options{MirrorProof: true}) returned an unexpected error: %v", err)
		}
		if len(got) != n*n {
			t.Fatalf("DCTDump(Options{MirrorProof: true}) returned %d coefficients, want %d (the full mirrorproof grid)", len(got), n*n)
		}
		want := mirrorproofVariant(base)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("DCTDump(Options{MirrorProof: true}) = %v..., want the _dct_mirrorproof variant of the instance's DCT %v... (Image::PHash dctdump with the mirrorproof option returns the _dct_mirrorproof transform: every other coefficient made absolute)",
				previewDCT(got), previewDCT(want))
		}
		if reflect.DeepEqual(got, base) {
			t.Errorf("DCTDump(Options{MirrorProof: true}) returned the plain base dump; the mirrorproof option must select the _dct_mirrorproof variant, not the base DCT (the fixture has %d negative odd-index coefficients, so the variant is observable)",
				countNegativeOddCoefficients(base))
		}
	})

	t.Run("mirror with mirrorproof errors like the Perl _mirroring croak", func(t *testing.T) {
		if got, err := p.DCTDump(phash.Options{Mirror: true, MirrorProof: true}); err == nil {
			t.Errorf("DCTDump(Options{Mirror: true, MirrorProof: true}) returned %d coefficients with no error; want an exclusivity error (Image::PHash dctdump's _mirroring croaks: options mirror and mirrorproof are mutually exclusive)",
				len(got))
		}
	})
}
