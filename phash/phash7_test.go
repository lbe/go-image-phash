package phash_test

import (
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// TestPHash7MatchesPHashWithGeometry7x7AndReduce verifies that PHash7 is the
// convenience shorthand for the 7x7 reduced hash on the same instance, the Go
// analogue of Image::PHash 0.3's pHash7 method (which calls pHash with
// geometry => '7x7', reduce => 1):
//
//   - on the same instance, PHash7().Hex() equals
//     PHash(Options{Geometry: "7x7", Reduce: true}).Hex(): whichever entry
//     point runs first primes the instance's cached resize/DCT, and the other
//     hashes the same coefficients, so the convenience form and the explicit
//     geometry/reduce options are interchangeable on one instance.
func TestPHash7MatchesPHashWithGeometry7x7AndReduce(t *testing.T) {
	p, err := phash.NewFromImage(syntheticFixtureImage(t, loadSyntheticGolden(t)), 0, phash.Options{})
	if err != nil {
		t.Fatalf("NewFromImage(synthetic image, 0, Options{}) returned an unexpected error: %v", err)
	}

	viaConvenience, err := p.PHash7()
	if err != nil {
		t.Fatalf("PHash7() returned an unexpected error: %v", err)
	}
	viaOptions, err := p.PHash(phash.Options{Geometry: "7x7", Reduce: true})
	if err != nil {
		t.Fatalf("PHash(Options{Geometry: \"7x7\", Reduce: true}) returned an unexpected error: %v", err)
	}

	if got, want := viaConvenience.Hex(), viaOptions.Hex(); got != want {
		t.Errorf("PHash7().Hex() = %q, want %q (PHash(Options{Geometry: \"7x7\", Reduce: true}).Hex() on the same instance); pHash7 is Image::PHash 0.3's convenience for geometry 7x7 with reduce on",
			got, want)
	}
}
