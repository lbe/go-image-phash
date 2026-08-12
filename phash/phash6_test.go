package phash_test

import (
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// TestPHash6MatchesPHashWithGeometry6x6ReduceAndMedian verifies that PHash6
// is the convenience shorthand for the 6x6 reduced median hash on the same
// instance, the Go analogue of Image::PHash 0.3's pHash6 method (which calls
// pHash with geometry => '6x6', reduce => 1, method => 'median'):
//
//   - on the same instance, PHash6().Hex() equals
//     PHash(Options{Geometry: "6x6", Reduce: true, Method: "median"}).Hex():
//     whichever entry point runs first primes the instance's cached
//     resize/DCT, and the other hashes the same coefficients, so the
//     convenience form and the explicit geometry/reduce/median options are
//     interchangeable on one instance.
func TestPHash6MatchesPHashWithGeometry6x6ReduceAndMedian(t *testing.T) {
	p, err := phash.NewFromImage(syntheticFixtureImage(t, loadSyntheticGolden(t)), 0, phash.Options{})
	if err != nil {
		t.Fatalf("NewFromImage(synthetic image, 0, Options{}) returned an unexpected error: %v", err)
	}

	viaConvenience, err := p.PHash6()
	if err != nil {
		t.Fatalf("PHash6() returned an unexpected error: %v", err)
	}
	viaOptions, err := p.PHash(phash.Options{Geometry: "6x6", Reduce: true, Method: "median"})
	if err != nil {
		t.Fatalf("PHash(Options{Geometry: \"6x6\", Reduce: true, Method: \"median\"}) returned an unexpected error: %v", err)
	}

	if got, want := viaConvenience.Hex(), viaOptions.Hex(); got != want {
		t.Errorf("PHash6().Hex() = %q, want %q (PHash(Options{Geometry: \"6x6\", Reduce: true, Method: \"median\"}).Hex() on the same instance); pHash6 is Image::PHash 0.3's convenience for geometry 6x6 with reduce and median",
			got, want)
	}
}
