package phash_test

import (
	"image"
	"image/color"
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// TestReducedImageResizesOnDemandToConfiguredSizeAndNeverReferencesAPutPoolBuffer
// verifies the ReducedImage helper, the Go analogue of Image::PHash 0.3's
// reducedimage method (which runs the reduce step unless $self->{im_scaled}
// is already set and then returns $self->{im_scaled}):
//
//   - the resize happens on demand: a fresh instance over a 5×5 source with a
//     configured resize of 16 returns, without any prior hash call, an image
//     whose Bounds are exactly (0, 0, 16, 16) and whose pixels are exactly
//     the default scaler's output for that source and size — the same scaled
//     image the DCT step hashes — not the original 5×5 source;
//   - once scaled, a later ReducedImage call returns the same cached image
//     instead of resizing again (Perl's "Non rescaled" identity: the second
//     call returns $self->{im_scaled} without re-running the reduce step);
//   - the returned image never references a pool buffer that has been Put:
//     after a second instance resizes a very different image through the
//     default pool, every pixel of the first instance's returned image is
//     unchanged. ReducedImage may hold the pooled buffer checked out for the
//     instance lifetime or copy the scaled pixels into owned storage before
//     release, but handing out a buffer that was already returned to the pool
//     shows up as the pool's next borrower overwriting the image.
func TestReducedImageResizesOnDemandToConfiguredSizeAndNeverReferencesAPutPoolBuffer(t *testing.T) {
	t.Run("resizes on demand to the configured NxN", func(t *testing.T) {
		src := gradientFixture()
		const n = 16
		p, err := phash.NewFromImage(src, n, phash.Options{})
		if err != nil {
			t.Fatalf("NewFromImage(5x5 gradient fixture, %d, Options{}) returned an unexpected error: %v", n, err)
		}

		// No hash precedes this call: the scale must be triggered by
		// ReducedImage itself, on demand.
		scaled := p.ReducedImage()
		if scaled == nil {
			t.Fatalf("ReducedImage() returned a nil image; want the configured %dx%d scaled image", n, n)
		}
		if got, want := scaled.Bounds(), image.Rect(0, 0, n, n); !got.Eq(want) {
			t.Fatalf("ReducedImage().Bounds() = %v, want %v (the configured resize %dx%d; the 5x5 source is not already scaled, so the resize must run on this call and the source must not be returned unwrapped)", got, want, n, n)
		}

		// The returned image must be exactly what the DCT path hashes: the
		// default ApproxBiLinear Resize output for the same source and size.
		want, release := phash.Resize(src, n, nil)
		defer release()
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				if got := scaled.At(x, y); got != want.RGBAAt(x, y) {
					t.Errorf("ReducedImage() pixel (%d,%d) = %v, want %v (the scaled image used for DCT, i.e. the default Resize output for the same source and size)", x, y, got, want.RGBAAt(x, y))
				}
			}
		}
	})

	t.Run("later calls return the same scaled image without resizing again", func(t *testing.T) {
		const n = 16
		p, err := phash.NewFromImage(gradientFixture(), n, phash.Options{})
		if err != nil {
			t.Fatalf("NewFromImage(5x5 gradient fixture, %d, Options{}) returned an unexpected error: %v", n, err)
		}

		first := p.ReducedImage()
		if first == nil {
			t.Fatalf("first ReducedImage() returned a nil image")
		}
		again := p.ReducedImage()
		if again == nil {
			t.Fatalf("second ReducedImage() returned a nil image; want the cached %dx%d scaled image", n, n)
		}
		if again != first {
			t.Errorf("second ReducedImage() returned a different image than the first; want the same cached image (Perl's reducedimage returns $self->{im_scaled} once the reduce step has run, so the resize must happen only until the instance is scaled, not on every call)")
		}
	})

	t.Run("returned image never references a pool buffer that has been Put", func(t *testing.T) {
		const n = 32
		a, err := phash.NewFromImage(gradientFixture(), 0, phash.Options{})
		if err != nil {
			t.Fatalf("NewFromImage(5x5 gradient fixture, 0, Options{}) returned an unexpected error: %v", err)
		}
		scaled := a.ReducedImage()
		if scaled == nil {
			t.Fatalf("ReducedImage() returned a nil image")
		}
		if got, want := scaled.Bounds(), image.Rect(0, 0, n, n); !got.Eq(want) {
			t.Fatalf("ReducedImage().Bounds() = %v, want %v (default resize %dx%d)", got, want, n, n)
		}

		// Snapshot every pixel of the returned image, then let a second
		// instance over a fully black image resize through the default pool:
		// if the returned image referenced a buffer that was Put back to the
		// pool, that call reuses it — the pool's reset zeroes the canvas
		// before the scaler draws black — and the snapshot no longer matches.
		snapshot := make([]color.RGBA, n*n)
		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				snapshot[y*n+x] = color.RGBAModel.Convert(scaled.At(x, y)).(color.RGBA)
			}
		}

		b, err := phash.NewFromImage(blackFixture(n), 0, phash.Options{})
		if err != nil {
			t.Fatalf("NewFromImage(black image, 0, Options{}) returned an unexpected error: %v", err)
		}
		if _, err := b.PHash(phash.Options{}); err != nil {
			t.Fatalf("PHash(Options{}) on the black image returned an unexpected error: %v", err)
		}

		for y := 0; y < n; y++ {
			for x := 0; x < n; x++ {
				if got := color.RGBAModel.Convert(scaled.At(x, y)).(color.RGBA); got != snapshot[y*n+x] {
					t.Errorf("ReducedImage() pixel (%d,%d) changed from %v to %v after another instance resized into the pool; the returned image must not reference a pool buffer that has been Put (hold the checkout for the instance lifetime or copy the scaled pixels into owned storage before release)",
						x, y, snapshot[y*n+x], got)
				}
			}
		}
	})
}
