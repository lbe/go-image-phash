package phash

// HashFromDCT runs the full bit pipeline over a row-major DCT coefficient
// vector, turning it and the given Options into a Hash, the Go analogue of
// Image::PHash 0.3's pHash method applied to an already-computed DCT matrix:
// option validation (including the mutually exclusive mirror/mirrorproof
// pair), the optional mirror or mirrorproof DCT transform, the geometry /
// reduce / method coefficient selection with its extras and skip flag, the
// threshold conversion to bits, and the hex conversion of the resulting Hash.
//
// The caller's DCT vector is never mutated: the mirror and mirrorproof
// transforms return fresh slices and the selection and threshold steps only
// read, so coeffs is byte-for-byte identical after the call.
func HashFromDCT(coeffs []float64, resize int, opts Options) (Hash, error) {
	opts, resize, err := normalizeHashOptions(opts, resize)
	if err != nil {
		return Hash{}, err
	}
	return hashFromDCT(coeffs, resize, opts), nil
}

// hashFromDCT runs the bit pipeline steps of HashFromDCT over options
// normalizeHashOptions has already validated and defaulted: the optional
// mirror or mirrorproof DCT transform, the geometry / reduce / method
// coefficient selection with its extras and skip flag, and the threshold
// conversion to bits. PrintBitMatrixFromDCT shares this core so its
// validation runs once, not twice, and its mirror/resize defaults always
// match HashFromDCT's.
func hashFromDCT(coeffs []float64, resize int, opts Options) Hash {
	switch {
	case opts.Mirror:
		coeffs = mirrorDCT(coeffs)
	case opts.MirrorProof:
		coeffs = mirrorproofDCT(coeffs)
	}

	primary, extra := selectGeometry(coeffs, opts.Geometry, opts.Reduce, opts.Method, resize)
	skip := skipFlag(opts.Geometry, opts.Reduce)
	return NewHash(thresholdMethods[opts.Method](primary, skip, extra))
}

// thresholdMethods dispatches each Image::PHash 0.3 method name to the
// function that converts the selected primary and extra coefficients into a
// bit string. Validate guarantees the method is one of these five names, so
// the lookup always succeeds after HashFromDCT's validation step.
var thresholdMethods = map[string]func(primary []float64, skip int, extra []float64) string{
	"average":   applyAverage,
	"median":    applyMedian,
	"average_x": applyAverageX,
	"log":       applyLog,
	"diff":      applyDiff,
}

// skipFlag reports Image::PHash's $skip flag: 1 for a non-reduce NxN grid
// geometry (so DC, primary[0], is excluded from the threshold but still
// produces a bit) and 0 for the reduce and bit-count paths.
func skipFlag(geometry string, reduce bool) int {
	if reduce {
		return 0
	}
	if isGridGeometry(geometry) {
		return 1
	}
	return 0
}
