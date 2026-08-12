package phash_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// m31JPGPath returns the absolute path of the checked-in _examples/M31.jpg
// fixture, resolved against the test's working directory (the phash package
// dir), so NewFromPath is exercised the way a caller at the module root
// would. The fixture is a real JPEG, which is exactly why the package under
// test must have registered a jpeg decoder.
func m31JPGPath(t *testing.T) string {
	t.Helper()
	path, err := filepath.Abs("../_examples/M31.jpg")
	if err != nil {
		t.Fatalf("resolve absolute path of ../_examples/M31.jpg: %v", err)
	}
	return path
}

// TestNewFromPathLoadsImageFileOrReturnsLoadError exercises the path-based
// constructor end to end, covering the cycle acceptance criteria:
//
//   - decode formats are registered: the test package deliberately does NOT
//     blank-import image/jpeg itself (registration must come from the phash
//     package or the callers below get "image: unknown format" and
//     NewFromPath on the checked-in _examples/M31.jpg fails), so a passing
//     load of the JPEG proves the package registered its decoders;
//   - NewFromPath on _examples/M31.jpg with empty/zero Options (Image::PHash
//     0.3 defaults: resize 32, geometry 8x8, method average) returns a PHash
//     whose PHash(Options{}) call succeeds and produces the non-empty
//     64-bit (16 hex digit) perceptual hash of the file, like the Perl
//     image-object constructor's load;
//   - a missing path returns a load error rather than a silent nil or zero
//     hash;
//   - the constructor rejects resize <= 5 exactly like NewFromImage /
//     Image::PHash's "resize > 5 expected." croak, while a non-positive
//     resize still defaults to 32 and any resize > 5 is accepted.
func TestNewFromPathLoadsImageFileOrReturnsLoadError(t *testing.T) {
	path := m31JPGPath(t)

	t.Run("loads _examples/M31.jpg and hashes it", func(t *testing.T) {
		p, err := phash.NewFromPath(path, 0, phash.Options{})
		if err != nil {
			t.Fatalf("NewFromPath(%q, 0, Options{}) returned an unexpected error: %v", path, err)
		}
		if p == nil {
			t.Fatalf("NewFromPath(%q, 0, Options{}) returned a nil PHash", path)
		}
		h, err := p.PHash(phash.Options{})
		if err != nil {
			t.Fatalf("PHash(Options{}) on the loaded image returned an unexpected error: %v", err)
		}
		hex := h.Hex()
		if hex == "" {
			t.Errorf("PHash(Options{}).Hex() on _examples/M31.jpg is empty; want a non-empty perceptual hash under the Image::PHash 0.3 default settings (resize 32, geometry 8x8, method average)")
		}
		if len(hex) != 16 {
			t.Errorf("PHash(Options{}).Hex() on _examples/M31.jpg has %d hex digits (%q), want 16 for the 64-bit hash produced by the default 8x8 geometry", len(hex), hex)
		}
	})

	t.Run("undecodable file returns a load error", func(t *testing.T) {
		bad := filepath.Join(t.TempDir(), "not-an-image.dat")
		if err := os.WriteFile(bad, []byte("not a valid image"), 0o644); err != nil {
			t.Fatal(err)
		}
		p, err := phash.NewFromPath(bad, 0, phash.Options{})
		if err == nil {
			t.Fatalf("NewFromPath(%q) returned PHash %v with no error; want a decode error", bad, p)
		}
	})

	t.Run("missing path returns a load error", func(t *testing.T) {
		missing := filepath.Join(filepath.Dir(path), "no-such-image.jpg")
		p, err := phash.NewFromPath(missing, 0, phash.Options{})
		if err == nil {
			t.Errorf("NewFromPath(%q, 0, Options{}) returned PHash %v with no error; want a load error for a missing file", missing, p)
		}
	})

	t.Run("constructor rejects resize <= 5 like Image::PHash", func(t *testing.T) {
		for _, bad := range []int{1, 5} {
			if p, err := phash.NewFromPath(path, bad, phash.Options{}); err == nil {
				t.Errorf("NewFromPath(%q, %d, Options{}) returned PHash %v with no error; want a resize rejection (Image::PHash: resize > 5 expected)", path, bad, p)
			}
		}
		for _, ok := range []int{0, 6} {
			if _, err := phash.NewFromPath(path, ok, phash.Options{}); err != nil {
				t.Errorf("NewFromPath(%q, %d, Options{}) returned an unexpected error: %v; want acceptance (non-positive resize defaults to 32, and any resize > 5 is allowed)", path, ok, err)
			}
		}
	})
}
