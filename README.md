# go-image-phash

[![Go Reference](https://pkg.go.dev/badge/github.com/lbe/go-image-phash.svg)](https://pkg.go.dev/github.com/lbe/go-image-phash)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/Go-1.26.5-blue.svg)](https://go.dev/dl/)
[![Release](https://github.com/lbe/go-image-phash/actions/workflows/releases.yml/badge.svg)](https://github.com/lbe/go-image-phash/actions/workflows/releases.yml)
[![CI](https://github.com/lbe/go-image-phash/actions/workflows/ci.yml/badge.svg)](https://github.com/lbe/go-image-phash/actions/workflows/ci.yml)
[![codecov](https://codecov.io/gh/lbe/go-image-phash/branch/main/graph/badge.svg)](https://codecov.io/gh/lbe/go-image-phash)

DCT-based perceptual image hashing for Go. The hash algorithm matches
[Image::PHash](https://github.com/dkechag/Image-PHash) 0.3: same geometry /
reduce / method / mirror pipeline, reduced hashes, Hamming distance, and
bits-to-hex conversion, given the same luminance input.

Images are loaded with Go's `image` package and resized with
`golang.org/x/image/draw` (`ApproxBiLinear`). That means file-level hashes can
differ from Perl when the source is larger than the DCT size — Perl's Imager
`mixing` resize is not replicated. The bit math itself is the same.

```bash
go get github.com/lbe/go-image-phash/phash
```

There is also an exported [`dct`](dct/) package if you want the 2D DCT
directly (Math::DCT-style).

## Quick start

```go
package main

import (
	"fmt"
	"log"

	"github.com/lbe/go-image-phash/phash"
)

func main() {
	// Load an image. Default resize is 32x32, ready for the DCT step.
	p, err := phash.NewFromPath("photo.jpg", 0, phash.Options{})
	if err != nil {
		log.Fatal(err)
	}

	// Standard pHash: top 8x8 of the DCT → 64 bits, 16 hex chars.
	h, err := p.PHash(phash.Options{}) // geometry "8x8", method "average"
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(h.Hex())

	// Better 64-bit hash: the 64 lowest-frequency DCT values, not a flat 8x8.
	// Usually recommended over the default.
	h, err = p.PHash(phash.Options{Geometry: "64"})
	if err != nil {
		log.Fatal(err)
	}

	// Same idea with the log threshold (lower false-negative rate).
	h, err = p.PHash(phash.Options{Geometry: "64", Method: "log"})
	if err != nil {
		log.Fatal(err)
	}

	// Reduced hashes for indexing / cutting false positives.
	// PHash7: upper-left half of 7x7 → 27 bits, 7 hex chars.
	p7, err := p.PHash7()
	// Same as: p.PHash(phash.Options{Geometry: "7x7", Reduce: true})

	// PHash6: upper-left half of 6x6 with median → 20 bits, 5 hex chars.
	p6, err := p.PHash6()
	// Same as: p.PHash(Options{Geometry: "6x6", Reduce: true, Method: "median"})

	diff, err := phash.Distance(h, p7)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(diff)

	_ = p6
}
```

From an already-decoded image:

```go
p, err := phash.NewFromImage(img, 32, phash.Options{})
```

## How it works

1. Resize to N×N (default 32).
2. Convert to luminance: `r*0.3 + g*0.59 + b*0.11`.
3. 2D DCT on that grid.
4. Keep the low-frequency coeffs according to `geometry` / `reduce`.
5. Threshold to bits with the chosen `method`, then hex-encode.

Most of the wall time is decode and resize. Once you have a 32×32 luminance
block, the DCT and bit pipeline are cheap. The DCT lives in the exported
`dct` package (ported from Math::DCT).

Resize and DCT are cached on the first hash call for a given `PHash`
instance, so `PHash6`, `PHash7`, mirror variants, and option changes after that
are essentially free.

## Constructors

### `NewFromPath`

```go
p, err := phash.NewFromPath(path string, resize int, opts Options) (*PHash, error)
```

Opens and decodes a file (JPEG/PNG/GIF and whatever else the package registers
via blank imports). Load failures return an error — not a silent nil like some
Perl image backends.

- `resize` — DCT size. `0` or negative means 32. Must be greater than 5
  (same rule as Perl's `resize > 5 expected`).
- `opts` — kept for API shape; only `resize` matters at construct time. Pass
  hash options to `PHash` itself.

### `NewFromImage`

```go
p, err := phash.NewFromImage(img image.Image, resize int, opts Options) (*PHash, error)
```

Same as Perl's image-object constructor: hand it any `image.Image`.

## `PHash` options

```go
h, err := p.PHash(phash.Options{
	Geometry:    "8x8",     // DCT window; "8x8" is the classic 64-bit hash
	Reduce:      false,     // keep only the upper-left triangle of that window
	Method:      "average", // how DCT values become bits
	Mirror:      false,     // hash of the horizontally flipped image
	MirrorProof: false,     // hash that ignores horizontal flips
})
```

Call `.Hex()` for the hex string.

### geometry

Either:

- `"NxN"` — take the top-left N×N of the DCT. Default `"8x8"` → 64-bit hash.
- A bit count like `"64"` — walk the matrix in increasing diagonals and take
  that many lowest-frequency coeffs. A geometry of `64` still yields 64 bits,
  but from the upper-left half of an 11×11 corner instead of a flat 8×8. That
  usually spreads non-similar images farther apart and is the better default
  for many workloads.

### reduce

For square geometries only: keep bits on or above the `0,N → N,0` diagonal,
dropping the DC cell `(0,0)`. You get `(N-1)*(N+2)/2` bits.

An 8×8 reduced hash is 35 bits — shorter than a full 6×6, and often stronger
than a full 7×7 (49 bits). Bit-count geometries already behave a lot like
`reduce`, so the flag does not apply there.

### method

How DCT values become a bitmask. Default `"average"`.

| Method      | Behaviour                                                              | Trade-offs                                                                                                                                |
| ----------- | ---------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------------------------------- |
| `average`   | Compare each coeff to the mean                                         | Best similarity recognition; more false positives                                                                                         |
| `median`    | Threshold at the median                                                | Fewer collisions; more false negatives — useful for short/reduced hashes                                                                  |
| `average_x` | Average over the whole N×N (or 2×X next coeffs for bit-count geometry) | Near-`log` collisions, often slightly fewer false negatives; for reduced hashes                                                           |
| `log`       | Logarithmic average threshold                                          | Among the lowest collision rates; a bit more false negatives than `average`/`diff`, usually fewer than `median`; works on full hashes too |
| `diff`      | Bit from each coeff vs the previous                                    | Sits between `average` and `median` on most tests                                                                                         |

### mirror

Hash of the horizontally flipped image. Applied after the DCT, so you can
compute both the normal and mirrored hashes without a second resize/DCT.
Mutually exclusive with `MirrorProof`.

### mirrorproof

A hash that treats a mirrored pair as the same (or very close). Handy when
you only care that two photos are similar, including flips. If you need to
_detect_ that a flip happened, use `mirror` instead.

It costs roughly two bits of entropy, so prefer `mirror` unless you have a
reason for the resistant form.

## Indexing with PHash6 / PHash7

These reduced hashes are not meant to stand alone as comparison hashes —
alone they produce too many false positives. They shine in two places:

1. **Cut false positives** from a full-hash match. With a sensible threshold
   (around distance 2 for PHash6, 3 for PHash7), each can drop false positives
   by 60–90%. Using both can get you past 95%.
2. **Cheap DB indexes** when you only need to catch simple edits (resize,
   recompress) or can tolerate some misses. Store PHash6/PHash7 as indexed
   columns, then confirm with a full-hash distance:

```sql
SELECT *
FROM hash_table
WHERE (phash7 = @phash7
       OR phash6 = @phash6)
  AND BIT_COUNT(CAST(CONV(phash, 16, 10) AS UNSIGNED)
              ^ CAST(CONV(@phash, 16, 10) AS UNSIGNED)) < 8;
```

### `PHash7`

```go
p7, err := p.PHash7()
// = PHash(Options{Geometry: "7x7", Reduce: true})
```

Good when the database cannot index on Hamming distance: many near-matches of
the full hash still share the same PHash7. For simple resize/compression,
expect around 98% of true matches to land in the lookup. Also useful as a
second check when two different photos somehow collide on the full hash but
not on the reduced one.

### `PHash6`

```go
p6, err := p.PHash6()
// = PHash(Options{Geometry: "6x6", Reduce: true, Method: "median"})
```

Same idea as PHash7, but looser: more hits, more noise. The two are meant to
complement each other — together they index and verify better than either
alone.

## Helpers

### On a `PHash` instance

```go
img := p.ReducedImage()                          // scaled N×N image (Perl reducedimage)
coeffs, err := p.DCTDump(opts)                   // full DCT grid; respects Mirror / MirrorProof
matrix, err := p.PrintBitMatrix(opts)            // printable 0/1 grid for the same options
```

`PrintBitMatrix` uses `Options.Separator` and `Options.Filler` for layout
(defaults: empty separator, space filler — same as Perl).

### Package functions

```go
diff, err := phash.Distance(a, b)                // Hamming distance (Perl diff)
hex := phash.BitsToHex(bits)                     // bits → uppercase hex (Perl b2h)

h, err := phash.HashFromLuminance(luma, n, opts) // already have a luminance block
h, err := phash.HashFromDCT(coeffs, n, opts)     // already have DCT coeffs
s, err := phash.PrintBitMatrixFromDCT(coeffs, n, opts)
grey := phash.Luminance(rgba, n)                 // RGBA → grayscale floats
```

`Distance` requires equal bit lengths.

### `Hash`

```go
h := phash.NewHash(bits)
h.Hex()               // uppercase hex
h.BitLen()            // exact bit count (need not be a multiple of 4)
h.IsZero()
v, err := h.Uint64()  // only when BitLen() ≤ 64
```

## CLI

```bash
go build -o phash ./cmd/phash
./phash -path photo.jpg
./phash -path photo.jpg -method log
```

`-path` is required. `-method` is one of `average`, `median`, `average_x`,
`log`, `diff` (default `average`).

## Parity with Image::PHash

|                                                                 |                                                        |
| --------------------------------------------------------------- | ------------------------------------------------------ |
| Bit pipeline (geometry, reduce, methods, mirror, hex, distance) | Matches 0.3 for the same luminance / DCT input         |
| Luminance                                                       | `r*0.3 + g*0.59 + b*0.11`                              |
| Resize                                                          | `ApproxBiLinear` into a pooled N×N `*image.RGBA`       |
| File → hash                                                     | Can differ from Perl when the image is larger than N×N |
| Perl image backends                                             | Not supported — no Imager / GD / ImageMagick switch    |

If you care about stable hashes across deployments, pin this module version
and keep resize/options fixed.

### Choosing a distance threshold

There is no universal cutoff. It depends on the images and how many false
positives you can stand.

- Pairwise "are these two similar?": a >1% false-positive rate may be fine;
  distance above ~10 can work.
- Deduping a large collection: keep false positives near zero.

Rough guidance from Image::PHash:

- Full 64-bit pHash: distance **3–7** keeps false positives close to 0%.
- PHash7 / PHash6 as lookups: usually no more than **3** and **2**. Alone they
  still throw a lot of false positives — use them with the full hash.

Do not reuse [pHash.org](https://www.phash.org/) thresholds here. That library
prefilters with a 7×7 mean and always thresholds with median, and its distances
run about 50% higher on average (e.g. 7 here ≈ 11 there).

## Testing

```bash
make test       # hermetic goldens under testdata/
make test-all   # also runs -tags=perl cross-checks when Image::PHash is installed
make bench      # dct, gensyncpool, and phash benchmarks
```

Recorded numbers for the current toolchain live in [`docs/benchmarks.txt`](docs/benchmarks.txt).
Regenerate after a Go upgrade with `make bench-update`.

With Perl and Image::PHash 0.3 available, `go test -tags=perl ./...` checks
DCT, luminance hashes, the bit pipeline, `b2h`, and `diff` against the live
module. Without Perl, those tests skip.

## Layout

```text
github.com/lbe/go-image-phash
├── phash/                 # public API
├── dct/                   # Math::DCT-style transforms
├── internal/gensyncpool/
├── cmd/phash/
├── _examples/
└── testdata/
```

## Acknowledgements

Algorithm and API follow [Image::PHash](https://github.com/dkechag/Image-PHash)
by Dimitrios Kechagias (SpareRoom). The DCT code is ported from Math::DCT,
written for that module.

## License

MIT — see [LICENSE](LICENSE). Image::PHash is under the same terms as Perl 5.
