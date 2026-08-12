package phash

import "strconv"

// selectGeometry returns the primary and extra coefficient lists selected
// from a row-major DCT matrix (length resize*resize), mirroring the
// Image::PHash 0.3 selection order for the given geometry, reduce flag,
// method, and resize.
//
// A plain bit-count geometry (an integer K) selects the K lowest-frequency
// coefficients by walking the matrix in increasing anti-diagonals starting
// at i=1, so DC (index 0) is never selected; reduce is ignored, as in Perl.
//
// For NxN geometry the primary list holds the upper-left w*h block of
// coefficients in row-major order, or the reduced set when reduce is set
// (see selectReducedGrid). When the method is median or average_x, extra
// collects the coefficients Image::PHash also feeds to the threshold: the
// reduce path's dropped lower-right half, or for a non-reduce geometry with
// w < resize the two neighbors of the block at indices w+1 and (w+1)*resize.
func selectGeometry(dct []float64, geometry string, reduce bool, method string, resize int) (primary, extra []float64) {
	if bits, ok := parseBitCount(geometry); ok {
		return selectBitCount(dct, bits, method, resize)
	}

	w, h, ok := parseGridGeometry(geometry)
	if !ok {
		return nil, nil
	}

	if reduce {
		return selectReducedGrid(dct, w, h, method, resize)
	}

	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			primary = append(primary, dct[row*resize+col])
		}
	}

	if methodWantsExtras(method) && w < resize {
		extra = neighborExtras(dct, w, resize)
	}

	return primary, extra
}

// selectReducedGrid selects the coefficients Image::PHash 0.3's pHash reduce
// branch feeds to the threshold for an NxN geometry. Cells are visited in
// row-major order over the block but DC (i+j==0, index 0) is skipped, only
// the cells on the upper-left side of the anti-diagonal (i+j < (w+h)/2)
// enter primary, and the dropped lower-right half (i+j >= (w+h)/2) goes to
// extra for the median/average_x methods only.
func selectReducedGrid(dct []float64, w, h int, method string, resize int) (primary, extra []float64) {
	threshold := (w + h) / 2
	for row := 0; row < h; row++ {
		for col := 0; col < w; col++ {
			diag := row + col
			if diag == 0 {
				continue
			}
			idx := row*resize + col
			if diag < threshold {
				primary = append(primary, dct[idx])
			} else if methodWantsExtras(method) {
				extra = append(extra, dct[idx])
			}
		}
	}
	return primary, extra
}

// methodWantsExtras reports whether the method consumes the extra coefficient
// list (Image::PHash's $x_mult flag): median and average_x feed both lists to
// the threshold while the remaining methods use only the primary list.
func methodWantsExtras(method string) bool {
	return method == "median" || method == "average_x"
}

// neighborExtras returns the two coefficients Image::PHash pushes for a
// non-reduce NxN median/average_x geometry: the row-major neighbor just
// right of the block's first row (index w+1) and the neighbor just below the
// block's first column (index (w+1)*resize).
func neighborExtras(dct []float64, w, resize int) []float64 {
	next := w + 1
	return []float64{dct[next], dct[next*resize]}
}

// parseBitCount parses a plain bit-count geometry (an integer like "10" or
// "64") and reports its value. It returns ok=false for anything that is not
// a well-formed bit count; the returned value is only meaningful when ok is
// true.
func parseBitCount(geometry string) (bits int, ok bool) {
	if !bitCountRe.MatchString(geometry) {
		return 0, false
	}
	// bitCountRe matches ASCII digits only; practical geometries are tiny.
	bits, _ = strconv.Atoi(geometry) //nolint:errcheck // digits validated by bitCountRe
	return bits, true
}

// selectBitCount selects the K lowest-frequency coefficients for a plain
// bit-count geometry, mirroring Image::PHash 0.3's pHash else-branch. The
// primary list holds the first K coefficients of the anti-diagonal walk; for
// the median and average_x methods extra holds the next K coefficients while
// the remaining methods collect no extras. When the walk cannot supply K
// coefficients, everything it collected becomes primary and extra is empty.
func selectBitCount(dct []float64, bits int, method string, resize int) (primary, extra []float64) {
	ordered := lowestFrequencyCoeffs(dct, resize, 2*bits)
	if len(ordered) <= bits {
		return ordered, nil
	}
	primary = ordered[:bits]
	if methodWantsExtras(method) {
		extra = ordered[bits:]
	}
	return primary, extra
}

// lowestFrequencyCoeffs returns up to count coefficients of Image::PHash's
// anti-diagonal walk over a row-major DCT matrix: for each diagonal i from 1
// to resize-1 it visits the cells (i-j, j) for j in 0..i, in increasing
// frequency order. Starting at i=1 skips DC (index 0). The walk stops once
// count coefficients have been collected, so when count exceeds what the
// matrix can supply the result is shorter than count.
func lowestFrequencyCoeffs(dct []float64, resize, count int) []float64 {
	coeffs := make([]float64, 0, min(count, len(dct)))
	for i := 1; i < resize && len(coeffs) < count; i++ {
		for j := 0; j <= i && len(coeffs) < count; j++ {
			coeffs = append(coeffs, dct[(i-j)*resize+j])
		}
	}
	return coeffs
}
