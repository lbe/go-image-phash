# Image::PHash algorithmic parity (Go) — Design

**Date:** 2026-08-09\
**Branch:** `addPHash`\
**Reference:** Image::PHash **0.3** at `../src/Image-PHash`\
**Approach:** Hash core first (luminance/DCT in → `Hash` out), then I/O and resize

---

## 1. Goal and non-goals

### Goal

Go library with **algorithmic parity** to **Image::PHash 0.3**: same DCT → bitmask → hex / `diff` behavior for equivalent luminance inputs, plus a single Go decode/resize path.

### In scope

- Exported `phash`: constructors from path and `image.Image`; `PHash` / `PHash6` / `PHash7`; options (geometry, reduce, method, mirror, mirrorproof); helpers (`Distance`, hex / `b2h`, DCT dump, print bit matrix, reduced image access); resize+DCT caching
- Exported `dct` (Math::DCT-style), kept as a public package
- `internal/gensyncpool` (copied from sfpg-go)
- Resize via `golang.org/x/image/draw` (`ApproxBiLinear`) + pooled `*image.RGBA` (same pattern as sfpg-go `internal/thumbnail` pHash path — not gallery/EXIF/MD5)
- Module path `github.com/lbe/go-image-phash`
- `cmd/phash` smoke CLI
- Goldens in normal tests; optional live Perl behind a build tag

### Out of scope

- Multi-library Perl backends / Magick / Imager knobs
- Bit-identical file→hash vs every Perl backend
- `github.com/anthonynsimon/bild`
- `transforms/` package
- Perl calling conventions (wantarray, auto-probe image libs)

---

## 2. Layout and layers

```text
github.com/lbe/go-image-phash
├── phash/           # public Image::PHash-facing API
├── dct/             # public Math::DCT-style transforms
├── internal/
│   └── gensyncpool/ # copied helper
├── cmd/phash/       # CLI smoke tool
├── _examples/       # sample images
└── testdata/        # golden vectors (+ optional perl helpers)
```

### Layers (build order intent)

1. **Bit pipeline (pure):** luminance `[]float64` (or DCT coeffs) → geometry / reduce / method / mirror → `Hash`
2. **DCT:** `phash` calls exported `dct`
3. **I/O + resize:** decode → scale to N×N with `draw.ApproxBiLinear` into pooled `*image.RGBA` → grayscale floats → layer 1
4. **CLI:** thin wrapper over `phash`

### Repo hygiene

- **Delete:** `transforms/`, root `main.go`, `bild` dependency
- **Keep:** `perl/` as local reference only (not required for default `go test`)

---

## 3. `phash` API

### Types

- `Hash` — hex-backed perceptual hash
  - `Hex()`, `BitLen()`, `IsZero()`, `Uint64() (uint64, error)` when bit length ≤ 64
  - `Distance(other Hash) (int, error)` and package `Distance(a, b Hash) (int, error)` (Hamming / Image::PHash::diff)
- `Options` — `Geometry` (bit count or `NxN`), `Reduce`, `Method` (`average` | `median` | `average_x` | `log` | `diff`), `Mirror`, `MirrorProof`
- `PHash` — object holding image / DCT cache (Perl `$iph`)
- `Resizer` — optional interface; default scales with `ApproxBiLinear` into pooled N×N `*image.RGBA`

### Constructors

- `NewFromPath(...)` — path + resize (default 32) + settings
- `NewFromImage(img image.Image, ...)` — Go analogue of 0.3 image-object loading
- Invalid settings → `error` (Perl croak)
- Load failure → `error` (not silent undef)

### Methods

- `PHash(opts Options) (Hash, error)`
- `PHash6()`, `PHash7()`
- `ReducedImage() image.Image`
- `DCTDump()` — expose DCT usefully (match Perl `dctdump` intent)
- `PrintBitMatrix(...)`
- **Caching:** first hash computes resize+DCT; later option variants reuse cache (including mirror / mirrorproof as in Perl)

### Package functions

- `Distance`, `BitsToHex` (`b2h`)

---

## 4. Testing and build order

### Perl reference

Image::PHash **0.3** + Math::DCT, sourced from `../src/Image-PHash` (and installed DCT deps when generating vectors).

### Default `go test ./...`

- Hermetic goldens under `testdata/`
- Prefer **shared luminance / DCT inputs** → expected hex / distances (isolates algorithm from resize)
- Port behavioral cases from Perl `t/util.t`, `error.t`, `reduced.t`, `phash.t` where they do not depend on a specific Perl imager
- Test files named `*_test.go`

### Optional `go test -tags=perl`

- Shell out to Perl to regenerate or cross-check vectors when Image::PHash is available
- Skip cleanly if Perl / module missing

### Build order

1. Module path + copy `internal/gensyncpool` + remove `bild` / `transforms` / root `main`
2. Pure bit pipeline + `Hash` / `Distance` / `BitsToHex` (TDD)
3. Wire `dct` into that pipeline; DCT goldens
4. Decode + `ApproxBiLinear` + pooled N×N `*image.RGBA` + constructors
5. Full option matrix, pHash6/7, helpers, caching
6. `cmd/phash` + file-level goldens / optional perl tag

---

## 5. Delivery process

1. Write this design to `docs/superpowers/specs/2026-08-09-image-phash-parity-design.md` and commit on `addPHash`.
2. Run **`tdd-plan`** → `.pi/tdd-plans/<branch>.yaml` cycles ordered per §4.
3. Run **`review-plan`** on that tdd-plan until it converges (or hits the iteration limit); do not start execution until this finishes.
4. Execute with **`tdd-orchestrator`** (preferably in **Pi**, where `tdd-*` agents exist).
5. Human merge decision when the plan completes; no force-push to `main`.

**Branch:** continue on `addPHash`.
