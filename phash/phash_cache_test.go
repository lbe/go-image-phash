package phash_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// paintSolid fills every pixel of img with the flat color c. Mutating the
// source image this way after a PHash's first hash is the probe that makes
// the resize/DCT cache observable from outside: an instance that recomputes
// resize+luminance+DCT on every call sees the new pixels, while an instance
// that cached the first call's results still hashes the original content.
func paintSolid(img *image.RGBA, c color.RGBA) {
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = c.R
		img.Pix[i+1] = c.G
		img.Pix[i+2] = c.B
		img.Pix[i+3] = c.A
	}
}

// TestPHashCachesResizeAndDCTAcrossOptionVariants verifies that one PHash
// instance computes the resize and DCT of its image once and reuses the
// cached result for every later option variant, the caching contract of
// Image::PHash 0.3's pHash method:
//
//   - the first PHash call over the synthetic 32x32 fixture matches the
//     checked-in Perl golden for the default 8x8 average settings;
//   - after the source image is mutated, later calls with a different
//     geometry, method, and mirror/mirrorproof combination still hash the
//     original content, which only holds if resize/DCT ran once and later
//     variants reuse the cached DCT (a recompute would see the new pixels);
//   - a reference instance over an unmutated copy of the same image defines
//     each variant's expected hash, so any in-place thresholding that
//     mutated the cached DCT between variants, or any reference to a pooled
//     resize buffer that was Put and overwritten by another instance, shows
//     up as a variant result that no longer matches its reference;
//   - another instance hashing a very different image in between forces the
//     resize pool to reuse buffers, so a cache that still referenced a Put
//     buffer would be corrupted before the final variant is checked.
func TestPHashCachesResizeAndDCTAcrossOptionVariants(t *testing.T) {
	golden := loadSyntheticGolden(t)
	orig := syntheticFixtureImage(t, golden)
	clone := syntheticFixtureImage(t, golden)

	// The instance under test shares the source that is mutated below; the
	// reference instance holds an unmutated copy and defines the expected
	// hash for every option variant over the original pixels.
	cached, err := phash.NewFromImage(orig, 0, phash.Options{})
	if err != nil {
		t.Fatalf("NewFromImage(synthetic image, 0, Options{}) returned an unexpected error: %v", err)
	}
	ref, err := phash.NewFromImage(clone, 0, phash.Options{})
	if err != nil {
		t.Fatalf("NewFromImage(clone, 0, Options{}) returned an unexpected error: %v", err)
	}

	// Prime the cache with the Perl golden default hash.
	def := golden.Cases[0]
	first, err := cached.PHash(phash.Options{})
	if err != nil {
		t.Fatalf("first PHash(Options{}) returned an unexpected error: %v", err)
	}
	if got := first.Hex(); got != def.Hex {
		t.Fatalf("first PHash(Options{}).Hex() = %q, want %q (Image::PHash 0.3 scalar-context golden)", got, def.Hex)
	}

	// Mutate the shared source so any recompute from it hashes visibly
	// different pixels. A fresh instance over the mutated source must differ
	// from the reference's expected variant hash, proving the variant
	// assertions below genuinely probe whether the DCT was reused.
	paintSolid(orig, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	recomputed, err := phash.NewFromImage(orig, 0, phash.Options{})
	if err != nil {
		t.Fatalf("NewFromImage(mutated image, 0, Options{}) returned an unexpected error: %v", err)
	}

	// variant1 differs from the priming call in method, geometry, and mirror.
	variant1 := phash.Options{Geometry: "16x16", Method: "median", Mirror: true}
	want1, err := ref.PHash(variant1)
	if err != nil {
		t.Fatalf("reference PHash(%+v) returned an unexpected error: %v", variant1, err)
	}
	if got, mutErr := recomputed.PHash(variant1); mutErr != nil {
		t.Fatalf("PHash(%+v) on the mutated source returned an unexpected error: %v", variant1, mutErr)
	} else if got.Hex() == want1.Hex() {
		t.Fatalf("mutated source PHash(%+v).Hex() = %q equals the original's %q; the probe needs pixels that hash differently", variant1, got.Hex(), want1.Hex())
	}

	got1, err := cached.PHash(variant1)
	if err != nil {
		t.Fatalf("PHash(%+v) after source mutation returned an unexpected error: %v", variant1, err)
	}
	if got1.Hex() != want1.Hex() {
		t.Errorf("PHash(%+v) after the source image was mutated = %q, want %q (the original content's hash); the first call's resize/DCT must be cached and reused by later option variants instead of recomputed from the changed source",
			variant1, got1.Hex(), want1.Hex())
	}

	// Another instance hashing a very different image reuses the pooled
	// resize buffers the first call returned, so a cache that retained a Put
	// buffer would be overwritten by these calls.
	other, err := phash.NewFromImage(blackFixture(golden.Resize), 0, phash.Options{})
	if err != nil {
		t.Fatalf("NewFromImage(black image, 0, Options{}) returned an unexpected error: %v", err)
	}
	if _, blackErr := other.PHash(phash.Options{}); blackErr != nil {
		t.Fatalf("PHash(Options{}) on the black image returned an unexpected error: %v", blackErr)
	}
	if _, diffErr := other.PHash(phash.Options{Geometry: "16x16", Method: "diff"}); diffErr != nil {
		t.Fatalf("PHash(Options{Geometry: \"16x16\", Method: \"diff\"}) on the black image returned an unexpected error: %v", diffErr)
	}

	// variant2 differs in bit-count geometry, method, and mirrorproof, and is
	// checked only after the pooled buffers were reused, so a corrupted cache
	// shows up here as well.
	variant2 := phash.Options{Geometry: "64", Method: "average_x", MirrorProof: true}
	want2, err := ref.PHash(variant2)
	if err != nil {
		t.Fatalf("reference PHash(%+v) returned an unexpected error: %v", variant2, err)
	}
	got2, err := cached.PHash(variant2)
	if err != nil {
		t.Fatalf("PHash(%+v) after the resize pool was reused returned an unexpected error: %v", variant2, err)
	}
	if got2.Hex() != want2.Hex() {
		t.Errorf("PHash(%+v) after another instance reused the pooled resize buffer = %q, want %q (the original content's hash); the cached DCT must be owned storage that no later resize overwrites, and hashing must copy selected coeffs before any in-place thresholding so the cache survives across option variants",
			variant2, got2.Hex(), want2.Hex())
	}
}
