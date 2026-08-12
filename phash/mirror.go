package phash

import "math"

// mirrorDCT transforms a row-major DCT coefficient vector the way
// Image::PHash 0.3's _dct_mirror does: starting with the first coefficient,
// every other coefficient (odd 0-based indices) is negated (see
// negateOddIndices) while the rest are kept as-is, and the result is a new
// slice leaving the input untouched.
func mirrorDCT(dct []float64) []float64 {
	out := copyDCT(dct)
	negateOddIndices(out)
	return out
}

// negateOddIndices negates in place every element of v at an odd 0-based
// index, leaving even-indexed elements (including v[0]) untouched.
func negateOddIndices(v []float64) {
	for i := 1; i < len(v); i += 2 {
		v[i] = -v[i]
	}
}

// mirrorproofDCT transforms a row-major DCT coefficient vector the way
// Image::PHash 0.3's _dct_mirrorproof does: starting with the first
// coefficient, every other coefficient (odd 0-based indices) is replaced by
// its absolute value (see absOddIndices) while the rest are kept as-is, and
// the result is a new slice leaving the input untouched.
func mirrorproofDCT(dct []float64) []float64 {
	out := copyDCT(dct)
	absOddIndices(out)
	return out
}

// copyDCT returns a new slice of the same length and contents as v.
func copyDCT(v []float64) []float64 {
	out := make([]float64, len(v))
	copy(out, v)
	return out
}

// absOddIndices replaces in place every element of v at an odd 0-based index
// with its absolute value, leaving even-indexed elements (including v[0])
// untouched.
func absOddIndices(v []float64) {
	for i := 1; i < len(v); i += 2 {
		v[i] = math.Abs(v[i])
	}
}
