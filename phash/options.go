package phash

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

// Options controls the perceptual hash computation, mirroring the options
// accepted by Image::PHash 0.3's pHash method: geometry (either 'NxN' for a
// square matrix or a plain number of bits), reduce, method (average, median,
// average_x, log, diff), mirror, mirrorproof, and the printbitmatrix-only
// separator and filler (the strings printed around and instead of bit
// positions when PrintBitMatrix renders a matrix; the pipeline ignores them,
// like Perl's pHash which accepts but never reads them).
type Options struct {
	Geometry    string
	Reduce      bool
	Method      string
	Mirror      bool
	MirrorProof bool
	Separator   string
	Filler      string
}

const (
	// defaultGeometry and defaultMethod are Image::PHash 0.3's option
	// defaults, applied by Validate when the caller leaves the field empty.
	defaultGeometry = "8x8"
	defaultMethod   = "average"

	// defaultResize is Image::PHash's default DCT size, applied when the
	// caller passes a non-positive resize.
	defaultResize = 32
)

var (
	bitCountRe = regexp.MustCompile(`^\d+$`)
	gridRe     = regexp.MustCompile(`^(\d+)x(\d+)$`)

	validMethods = map[string]bool{
		"average":   true,
		"median":    true,
		"average_x": true,
		"log":       true,
		"diff":      true,
	}
)

// checkMirrorExclusive reports the error Image::PHash's _mirroring croaks on
// when opts sets both the mirror and mirrorproof options, which are mutually
// exclusive, and nil when they are compatible. Validate and DCTDump share it
// so the exclusivity rule and its message live in one place.
func checkMirrorExclusive(opts Options) error {
	if opts.Mirror && opts.MirrorProof {
		return errors.New("mirror and mirrorproof are mutually exclusive")
	}
	return nil
}

// parseGridGeometry parses an NxN geometry like "8x8" and reports its width
// and height. It returns ok=false for anything that is not a well-formed
// NxN grid; the returned dimensions are only meaningful when ok is true.
func parseGridGeometry(geometry string) (w, h int, ok bool) {
	m := gridRe.FindStringSubmatch(geometry)
	if m == nil {
		return 0, 0, false
	}
	// gridRe captures digit groups only; practical NxN sizes are tiny.
	w, _ = strconv.Atoi(m[1]) //nolint:errcheck // digits validated by gridRe
	h, _ = strconv.Atoi(m[2]) //nolint:errcheck // digits validated by gridRe
	return w, h, true
}

// isGridGeometry reports whether geometry is a well-formed NxN grid, the
// shape check PrintBitMatrixFromDCT and skipFlag both branch on.
func isGridGeometry(geometry string) bool {
	_, _, ok := parseGridGeometry(geometry)
	return ok
}

// Validate normalizes opts (applying the Image::PHash 0.3 defaults of
// geometry '8x8' and method 'average') and rejects illegal geometry shapes,
// unknown methods, NxN geometries larger than the resize value, and the
// mutually exclusive mirror/mirrorproof combination. A non-positive resize
// defaults to 32, Image::PHash's default DCT size, so the geometry-vs-resize
// check still applies.
func Validate(opts Options, resize int) (Options, error) {
	if opts.Geometry == "" {
		opts.Geometry = defaultGeometry
	}
	if opts.Method == "" {
		opts.Method = defaultMethod
	}

	if err := checkMirrorExclusive(opts); err != nil {
		return opts, err
	}

	if !validMethods[opts.Method] {
		return opts, fmt.Errorf("unknown method %q", opts.Method)
	}

	// A plain bit-count geometry has no shape or resize constraints.
	if bitCountRe.MatchString(opts.Geometry) {
		return opts, nil
	}

	w, h, ok := parseGridGeometry(opts.Geometry)
	if !ok {
		return opts, fmt.Errorf("invalid geometry %q", opts.Geometry)
	}
	if w != h {
		return opts, fmt.Errorf("geometry %q must be square", opts.Geometry)
	}
	if resize <= 0 {
		resize = defaultResize
	}
	if w > resize {
		return opts, fmt.Errorf("geometry %q exceeds resize %d", opts.Geometry, resize)
	}
	return opts, nil
}

// normalizeHashOptions validates opts like Validate and applies the resize
// default the bit pipeline assumes, returning the normalized options and the
// effective resize. HashFromDCT and PrintBitMatrixFromDCT share it so the
// validation and defaults live in one place and run once per call: the print
// path needs the effective resize for its triangle layout and the defaulted
// geometry to choose the grid branch, which is why it cannot just forward to
// HashFromDCT.
func normalizeHashOptions(opts Options, resize int) (Options, int, error) {
	opts, err := Validate(opts, resize)
	if err != nil {
		return Options{}, 0, err
	}
	if resize <= 0 {
		resize = defaultResize
	}
	return opts, resize, nil
}
