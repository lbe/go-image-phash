package phash

import "strings"

// PrintBitMatrixFromDCT renders the printable bit matrix for a row-major DCT
// coefficient vector and the given Options, the Go analogue of Image::PHash
// 0.3's printbitmatrix method applied to an already-computed DCT matrix: it
// runs the same option validation, mirror/mirrorproof transform, geometry /
// reduce / method selection, and threshold steps as HashFromDCT, then lays
// the resulting bits out as a print-friendly string — NxN geometries as
// rows/columns of '0'/'1' characters separated by opts.Separator with
// opts.Filler standing in for the reduced-out positions, and bit-count
// geometries as the anti-diagonal triangle Perl printbitmatrix builds, whose
// first position is the filler. Separator defaults to the empty string and
// filler to a single space, matching Perl's `separator || ”` and
// `filler || ' '` defaults.
func PrintBitMatrixFromDCT(coeffs []float64, resize int, opts Options) (string, error) {
	opts, resize, err := normalizeHashOptions(opts, resize)
	if err != nil {
		return "", err
	}
	hash := hashFromDCT(coeffs, resize, opts)
	if isGridGeometry(opts.Geometry) {
		return renderGridBits(hash.bits, opts), nil
	}
	return renderTriangleBits(hash.bits, opts, resize), nil
}

// renderGridBits lays out the bit string of a hashed NxN geometry as a grid
// of rows: each cell emits one '0'/'1' bit followed by the separator, except
// the reduced-out positions — the DC cell and the lower-right half beyond the
// anti-diagonal — which emit the filler followed by the separator, exactly
// Image::PHash 0.3's printbitmatrix grid branch. The bits are consumed in the
// row-major order the hash pipeline produced them, and every row ends with a
// newline. The separator and filler come from separatorAndFiller, so Perl's
// `separator || ”` and `filler || ' '` defaults apply when unset.
func renderGridBits(bits string, opts Options) string {
	w, h, _ := parseGridGeometry(opts.Geometry)
	threshold := (w + h) / 2
	sep, filler := separatorAndFiller(opts)
	var b strings.Builder
	idx := 0
	for i := 0; i < w; i++ {
		for j := 0; j < h; j++ {
			if opts.Reduce && (i+j == 0 || i+j >= threshold) {
				b.WriteString(filler)
			} else {
				b.WriteByte(bits[idx])
				idx++
			}
			b.WriteString(sep)
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// renderTriangleBits lays out the bit string of a hashed bit-count geometry
// as the anti-diagonal triangle Image::PHash 0.3's printbitmatrix builds:
// walking the upper triangle row by row, each visited cell appends the next
// bit (followed by the separator) to the row indexed by its column, starting
// with a first row whose first position is the filler, and stopping once the
// bits are exhausted. The consumption order is exactly the order the hash
// pipeline selected the bits, so the triangle reorders them by column, and
// the separator and filler default like Perl's `separator || ”` and
// `filler || ' '` when unset.
func renderTriangleBits(bits string, opts Options, resize int) string {
	sep, filler := separatorAndFiller(opts)
	rows := []string{filler + sep}
	idx := 0
outer:
	for i := 1; i < resize; i++ {
		for j := 0; j <= i; j++ {
			for len(rows) <= j {
				rows = append(rows, "")
			}
			rows[j] += string(bits[idx]) + sep
			idx++
			if idx >= len(bits) {
				break outer
			}
		}
	}
	return strings.Join(rows, "\n") + "\n"
}

// separatorAndFiller returns the strings printbitmatrix emits between and in
// place of bit positions for opts, applying Perl's `separator || ”` and
// `filler || ' '` defaults: the separator stays empty when unset while an
// unset filler becomes a single space, the characters Perl prints around and
// instead of bits.
func separatorAndFiller(opts Options) (sep, filler string) {
	sep = opts.Separator
	filler = opts.Filler
	if filler == "" {
		filler = " "
	}
	return sep, filler
}

// PrintBitMatrix renders the printable bit matrix for the instance's image
// and the given Options, the Go analogue of Image::PHash 0.3's printbitmatrix
// method: it renders the bits of the hash the instance's own DCT produces for
// opts — the same cached resize/DCT path PHash uses — as a print-friendly
// bit matrix string, exactly as PrintBitMatrixFromDCT renders an
// already-computed coefficient vector.
func (p *PHash) PrintBitMatrix(opts Options) (string, error) {
	return PrintBitMatrixFromDCT(p.dct(), p.resize, opts)
}
