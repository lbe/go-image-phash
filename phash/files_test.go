package phash_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// exampleImages are the checked-in _examples fixtures the file-level golden
// test hashes: M31_s.jpg is a scaled copy of M31.jpg, M31_th.jpg the 32x32
// thumbnail (already at the DCT resize, so the Go pipeline's resize is an
// identity), M31_mirr.jpg the horizontally mirrored copy, and SolarEclipse.jpg
// an unrelated image.
var exampleImages = []string{"M31.jpg", "M31_mirr.jpg", "M31_s.jpg", "M31_th.jpg", "SolarEclipse.jpg"}

// exampleImagePath resolves the absolute path of an _examples fixture against
// the test's working directory (the phash package dir), the same way a caller
// at the module root would reference the checked-in examples.
func exampleImagePath(t *testing.T, name string) string {
	t.Helper()
	path, err := filepath.Abs(filepath.Join("..", "_examples", name))
	if err != nil {
		t.Fatalf("resolve absolute path of ../_examples/%s: %v", name, err)
	}
	return path
}

// hashExampleImage loads the fixture through NewFromPath with a zero resize
// and empty/zero Options — Image::PHash 0.3 defaults (resize 32, geometry
// 8x8, method average) — and hashes it with opts, the exact file-level flow
// the goldens pin: decode (the jpeg format must be registered by the phash
// package itself; this test package deliberately does not blank-import
// image/jpeg), ApproxBiLinear resize, luminance, DCT, and the bit pipeline.
func hashExampleImage(t *testing.T, name string, opts phash.Options) phash.Hash {
	t.Helper()
	p, err := phash.NewFromPath(exampleImagePath(t, name), 0, phash.Options{})
	if err != nil {
		t.Fatalf("NewFromPath(_examples/%s, 0, Options{}) returned an unexpected error: %v", name, err)
	}
	h, err := p.PHash(opts)
	if err != nil {
		t.Fatalf("PHash(%+v) on _examples/%s returned an unexpected error: %v", opts, name, err)
	}
	return h
}

// filesGolden is the schema of testdata/files/files.json: the stable
// file-level hash goldens for the checked-in _examples images under the Go
// pipeline, plus the exact settings the goldens were produced with.
type filesGolden struct {
	About  string            `json:"about"`
	Hashes map[string]string `json:"hashes"`
}

// loadFilesGolden opens and decodes ../testdata/files/files.json, the golden
// file recording the stable file-level hashes of the _examples images.
func loadFilesGolden(t *testing.T) *filesGolden {
	t.Helper()
	f, err := os.Open("../testdata/files/files.json")
	if err != nil {
		t.Fatalf("open file-level hash goldens: %v", err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Errorf("close file-level hash goldens: %v", err)
		}
	})

	var golden filesGolden
	if err := json.NewDecoder(f).Decode(&golden); err != nil {
		t.Fatalf("decode file-level hash goldens: %v", err)
	}
	return &golden
}

// TestExampleImagesProduceStableFileLevelHashGoldens is the integration
// acceptance that the checked-in _examples images produce stable file-level
// hash goldens under the Go pipeline: hashing each image with empty/zero
// Options (Image::PHash 0.3 defaults: resize 32, geometry 8x8, method
// average) through the Go ApproxBiLinear resize + luminance + DCT pipeline
// must reproduce exactly the hex hash recorded in testdata/files/files.json,
// and the similar-pair distance thresholds must be derived from that same Go
// pipeline rather than copied blindly from Perl t/phash.t.
func TestExampleImagesProduceStableFileLevelHashGoldens(t *testing.T) {
	golden := loadFilesGolden(t)

	t.Run("hashes match the file goldens", func(t *testing.T) {
		for _, name := range exampleImages {
			got := hashExampleImage(t, name, phash.Options{}).Hex()
			want, ok := golden.Hashes[name]
			if !ok {
				t.Errorf("no golden hash recorded for _examples/%s in testdata/files/files.json; the Go pipeline produces %q under empty/zero Options — record it to pin the stable file-level golden", name, got)
				continue
			}
			if got != want {
				t.Errorf("_examples/%s under empty/zero Options (Image::PHash 0.3 defaults: resize 32, geometry 8x8, method average) via the Go ApproxBiLinear resize+luminance+DCT pipeline: got hex hash %q, want the file golden %q from testdata/files/files.json", name, got, want)
			}
		}
	})

	t.Run("similar pairs stay close under the Go pipeline", func(t *testing.T) {
		// Threshold derived from the Go pipeline's own distances: M31 vs
		// M31_s measures 11 and M31 vs M31_th measures 8, so a similar-pair
		// bound of < 16 holds with margin. Perl t/phash.t's < 8 is NOT
		// reused: it would fail here, because M31 vs M31_s is 11 under this
		// pipeline.
		const similarMax = 16
		pairs := []struct{ a, b string }{
			{"M31.jpg", "M31_s.jpg"},
			{"M31.jpg", "M31_th.jpg"},
		}
		for _, pair := range pairs {
			ha := hashExampleImage(t, pair.a, phash.Options{})
			hb := hashExampleImage(t, pair.b, phash.Options{})
			dist, err := phash.Distance(ha, hb)
			if err != nil {
				t.Fatalf("Distance(_examples/%s, _examples/%s): %v", pair.a, pair.b, err)
			}
			if dist >= similarMax {
				t.Errorf("Distance(_examples/%s, _examples/%s) = %d, want < %d (similar pair under the Go pipeline; threshold derived from the measured distances 11 and 8)", pair.a, pair.b, dist, similarMax)
			}
		}
	})

	t.Run("the mirrored image hashes close to the mirror option", func(t *testing.T) {
		// mirror(M31) vs plain(M31_mirr) measures 7 under the Go pipeline, so
		// a bound of < 16 holds with margin. Perl t/phash.t's < 4 is NOT
		// reused: it would fail here, because the Go distance is 7.
		const mirrorMax = 16
		ha := hashExampleImage(t, "M31.jpg", phash.Options{Mirror: true})
		hb := hashExampleImage(t, "M31_mirr.jpg", phash.Options{})
		dist, err := phash.Distance(ha, hb)
		if err != nil {
			t.Fatalf("Distance(mirror(M31), M31_mirr): %v", err)
		}
		if dist >= mirrorMax {
			t.Errorf("Distance(PHash(M31, mirror), PHash(M31_mirr)) = %d, want < %d (mirror-similar pair under the Go pipeline; threshold derived from the measured distance 7)", dist, mirrorMax)
		}
	})

	t.Run("an unrelated image stays far", func(t *testing.T) {
		// M31 vs SolarEclipse measures 32 under the Go pipeline (half the
		// bits differ), so > 16 holds with margin. The direction agrees with
		// Perl t/phash.t's > 16 but the threshold is confirmed against the Go
		// distances rather than copied.
		const dissimilarMin = 16
		ha := hashExampleImage(t, "M31.jpg", phash.Options{})
		hb := hashExampleImage(t, "SolarEclipse.jpg", phash.Options{})
		dist, err := phash.Distance(ha, hb)
		if err != nil {
			t.Fatalf("Distance(M31, SolarEclipse): %v", err)
		}
		if dist <= dissimilarMin {
			t.Errorf("Distance(_examples/M31.jpg, _examples/SolarEclipse.jpg) = %d, want > %d (dissimilar pair under the Go pipeline; measured distance is 32)", dist, dissimilarMin)
		}
	})
}
