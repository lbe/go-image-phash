package phash

import (
	"slices"
	"testing"
)

// TestSelectGeometrySelectsUpperLeftDCTCoefficientsAndExtras verifies that
// NxN geometry without reduce selects the upper-left N*N DCT coefficients in
// Image::PHash's row-major order (index i*resize+j), keeping DC in the
// primary list, and that the median/average_x paths also collect the Perl
// extras — including the special non-reduce NxN neighbor push of the
// coefficients at (xs+1) and ((xs+1)*resize) when xs < resize.
func TestSelectGeometrySelectsUpperLeftDCTCoefficientsAndExtras(t *testing.T) {
	const resize = 32
	dct := make([]float64, resize*resize)
	for i := range dct {
		dct[i] = float64(i)
	}

	// Perl selection for geometry "4x4", dct_size 32: dct[i*32+j] for
	// i,j in 0..3 in row-major order, plus neighbor extras dct[4+1] and
	// dct[(4+1)*32] for the median/average_x paths.
	fourByFour := []float64{
		0, 1, 2, 3,
		32, 33, 34, 35,
		64, 65, 66, 67,
		96, 97, 98, 99,
	}

	cases := []struct {
		name       string
		geometry   string
		method     string
		wantPrim   []float64
		wantExtras []float64
	}{
		{
			name:       "4x4 median keeps DC in primary and pushes neighbor extras",
			geometry:   "4x4",
			method:     "median",
			wantPrim:   fourByFour,
			wantExtras: []float64{5, 160},
		},
		{
			name:       "4x4 average_x pushes the same neighbor extras",
			geometry:   "4x4",
			method:     "average_x",
			wantPrim:   fourByFour,
			wantExtras: []float64{5, 160},
		},
		{
			name:       "4x4 average collects no extras",
			geometry:   "4x4",
			method:     "average",
			wantPrim:   fourByFour,
			wantExtras: nil,
		},
		{
			name:       "32x32 median selects the full matrix and skips the neighbor push when xs equals resize",
			geometry:   "32x32",
			method:     "median",
			wantPrim:   dct[:resize*resize],
			wantExtras: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			primary, extra := selectGeometry(dct, tc.geometry, false, tc.method, resize)
			if !slices.Equal(primary, tc.wantPrim) {
				t.Errorf("selectGeometry(%s, reduce=false, %s) primary = %v, want %v",
					tc.geometry, tc.method, primary, tc.wantPrim)
			}
			if !slices.Equal(extra, tc.wantExtras) {
				t.Errorf("selectGeometry(%s, reduce=false, %s) extra = %v, want %v",
					tc.geometry, tc.method, extra, tc.wantExtras)
			}
		})
	}
}

// TestSelectGeometryReduceDropsDCAndLowerRightHalf verifies that NxN
// geometry with reduce enabled mirrors Image::PHash 0.3's pHash NxN branch:
// cells are visited in row-major order over the block but DC (i+j==0, index
// 0) is skipped and only cells on the upper-left side of the anti-diagonal
// (i+j < (xs+ys)/2 = N) go into primary, so its length is
// (N-1)*(N+2)/2; the dropped lower-right half (i+j >= N) is pushed into
// extra for the median/average_x paths while the remaining methods collect
// no extras, and the non-reduce neighbor push never fires.
func TestSelectGeometryReduceDropsDCAndLowerRightHalf(t *testing.T) {
	const resize = 32
	dct := make([]float64, resize*resize)
	for i := range dct {
		dct[i] = float64(i)
	}

	// Perl reduce selection for geometry "4x4", dct_size 32: for i,j in
	// 0..3 in row-major order, skip (i,j) with i+j==0 (DC at index 0), push
	// dct[i*32+j] to @array while i+j < 4, and push the lower-right half
	// (i+j >= 4) to @extra for the median/average_x paths. These lists are
	// Perl-generated goldens; the reduce path must not add the non-reduce
	// neighbor extras (dct[5], dct[160]).
	wantFourByFourPrim := []float64{1, 2, 3, 32, 33, 34, 64, 65, 96}
	wantFourByFourExtra := []float64{35, 66, 67, 97, 98, 99}

	cases := []struct {
		name       string
		geometry   string
		method     string
		wantPrim   []float64
		wantExtras []float64
	}{
		{
			name:       "4x4 median drops DC and pushes the lower-right half into extra",
			geometry:   "4x4",
			method:     "median",
			wantPrim:   wantFourByFourPrim,
			wantExtras: wantFourByFourExtra,
		},
		{
			name:       "4x4 average_x pushes the same lower-right half extras",
			geometry:   "4x4",
			method:     "average_x",
			wantPrim:   wantFourByFourPrim,
			wantExtras: wantFourByFourExtra,
		},
		{
			name:     "4x4 average collects no extras",
			geometry: "4x4",
			method:   "average",
			wantPrim: wantFourByFourPrim,
		},
		{
			name:     "4x4 log collects no extras",
			geometry: "4x4",
			method:   "log",
			wantPrim: wantFourByFourPrim,
		},
		{
			name:     "4x4 diff collects no extras",
			geometry: "4x4",
			method:   "diff",
			wantPrim: wantFourByFourPrim,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			primary, extra := selectGeometry(dct, tc.geometry, true, tc.method, resize)
			if !slices.Equal(primary, tc.wantPrim) {
				t.Errorf("selectGeometry(%s, reduce=true, %s) primary = %v, want Perl reduced set %v",
					tc.geometry, tc.method, primary, tc.wantPrim)
			}
			if !slices.Equal(extra, tc.wantExtras) {
				t.Errorf("selectGeometry(%s, reduce=true, %s) extra = %v, want %v",
					tc.geometry, tc.method, extra, tc.wantExtras)
			}
		})
	}

	// Perl's 8x8 reduced hash holds (N-1)*(N+2)/2 = 35 bits, so primary
	// must have exactly that many coefficients and no DC; the lower-right
	// half (i+j >= 8) supplies the extras for the median path.
	t.Run("8x8 median selects the (N-1)*(N+2)/2 upper-left coefficients", func(t *testing.T) {
		primary, extra := selectGeometry(dct, "8x8", true, "median", resize)
		wantPrimLen := (8 - 1) * (8 + 2) / 2
		if len(primary) != wantPrimLen {
			t.Errorf("selectGeometry(8x8, reduce=true, median) primary has length %d, want %d (Perl reduced bit count)",
				len(primary), wantPrimLen)
		}
		if len(extra) != 8*8-1-wantPrimLen {
			t.Errorf("selectGeometry(8x8, reduce=true, median) extra has length %d, want %d (lower-right half)",
				len(extra), 8*8-1-wantPrimLen)
		}
		if len(primary) > 0 && primary[0] != dct[1] {
			t.Errorf("first reduced primary coefficient = %v, want dct[1] = %v (DC skipped, walk starts at (0,1))",
				primary[0], dct[1])
		}
		if slices.Contains(primary, dct[0]) || slices.Contains(extra, dct[0]) {
			t.Errorf("reduce must drop DC: primary contains dct[0] = %v? %t; extra contains dct[0]? %t",
				dct[0], slices.Contains(primary, dct[0]), slices.Contains(extra, dct[0]))
		}
	})
}

// TestSelectGeometryBitCountSelectsLowestFrequencyCoefficientsSkippingDC
// verifies that a plain bit-count geometry (an integer K) selects the K
// lowest-frequency DCT coefficients by walking the row-major matrix in
// increasing anti-diagonals starting at i=1, exactly like Image::PHash 0.3's
// pHash else-branch: DC (index 0) is never selected, the primary list holds
// the first K coefficients of that diagonal order, and for the $x_mult
// methods (median, average_x) extra collects the next K coefficients in the
// same walk while the remaining methods stop as soon as the primary list is
// full. Reduce is ignored for bit-count geometry, as in Perl.
func TestSelectGeometryBitCountSelectsLowestFrequencyCoefficientsSkippingDC(t *testing.T) {
	const resize = 32
	dct := make([]float64, resize*resize)
	for i := range dct {
		dct[i] = float64(i)
	}

	// Perl diagonal walk for geometry=K, dct_size 32: for i = 1..dct_size-1
	// and j = 0..i, take dct[(i-j)*32+j]. Starting the walk at i=1 skips DC
	// (index 0); the first primary coefficient is dct[32], the AC coefficient
	// just below DC. These lists are Perl-generated goldens.
	wantK10Primary := []float64{32, 1, 64, 33, 2, 96, 65, 34, 3, 128}
	wantK10Extra := []float64{97, 66, 35, 4, 160, 129, 98, 67, 36, 5}

	wantK64Primary := []float64{
		32, 1, 64, 33, 2, 96, 65, 34, 3, 128, 97, 66, 35, 4, 160,
		129, 98, 67, 36, 5, 192, 161, 130, 99, 68, 37, 6, 224, 193, 162,
		131, 100, 69, 38, 7, 256, 225, 194, 163, 132, 101, 70, 39, 8, 288,
		257, 226, 195, 164, 133, 102, 71, 40, 9, 320, 289, 258, 227, 196, 165,
		134, 103, 72, 41,
	}
	wantK64Extra := []float64{
		10, 352, 321, 290, 259, 228, 197, 166, 135, 104, 73, 42, 11, 384, 353,
		322, 291, 260, 229, 198, 167, 136, 105, 74, 43, 12, 416, 385, 354, 323,
		292, 261, 230, 199, 168, 137, 106, 75, 44, 13, 448, 417, 386, 355, 324,
		293, 262, 231, 200, 169, 138, 107, 76, 45, 14, 480, 449, 418, 387, 356,
		325, 294, 263, 232,
	}

	cases := []struct {
		name       string
		geometry   string
		reduce     bool
		method     string
		wantPrim   []float64
		wantExtras []float64
	}{
		{
			name:       "10 bits median walks diagonals from i=1 skipping DC and fills 10 extras",
			geometry:   "10",
			method:     "median",
			wantPrim:   wantK10Primary,
			wantExtras: wantK10Extra,
		},
		{
			name:       "10 bits average_x fills the same 10 extras",
			geometry:   "10",
			method:     "average_x",
			wantPrim:   wantK10Primary,
			wantExtras: wantK10Extra,
		},
		{
			name:     "10 bits average collects no extras",
			geometry: "10",
			method:   "average",
			wantPrim: wantK10Primary,
		},
		{
			name:     "10 bits log collects no extras",
			geometry: "10",
			method:   "log",
			wantPrim: wantK10Primary,
		},
		{
			name:     "10 bits diff collects no extras",
			geometry: "10",
			method:   "diff",
			wantPrim: wantK10Primary,
		},
		{
			name:       "10 bits median ignores the reduce flag like Perl",
			geometry:   "10",
			reduce:     true,
			method:     "median",
			wantPrim:   wantK10Primary,
			wantExtras: wantK10Extra,
		},
		{
			name:       "64 bits median selects 64 primary and 64 extra coefficients",
			geometry:   "64",
			method:     "median",
			wantPrim:   wantK64Primary,
			wantExtras: wantK64Extra,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			primary, extra := selectGeometry(dct, tc.geometry, tc.reduce, tc.method, resize)
			if !slices.Equal(primary, tc.wantPrim) {
				t.Errorf("selectGeometry(%s, reduce=%t, %s) primary = %v, want Perl diagonal walk %v",
					tc.geometry, tc.reduce, tc.method, primary, tc.wantPrim)
			}
			if !slices.Equal(extra, tc.wantExtras) {
				t.Errorf("selectGeometry(%s, reduce=%t, %s) extra = %v, want %v",
					tc.geometry, tc.reduce, tc.method, extra, tc.wantExtras)
			}
		})
	}

	// The walk starts at i=1, so DC (index 0) is never selected: the first
	// primary coefficient is dct[resize], the AC coefficient just below DC.
	t.Run("diagonal walk starts at i=1 so DC is skipped", func(t *testing.T) {
		primary, _ := selectGeometry(dct, "10", false, "median", resize)
		if len(primary) != 10 {
			t.Fatalf("selectGeometry(\"10\", reduce=false, median) primary has length %d, want 10", len(primary))
		}
		if primary[0] != dct[resize] {
			t.Errorf("first selected coefficient = %v, want dct[%d] = %v (walk starts at i=1, skipping DC)",
				primary[0], resize, dct[resize])
		}
		if slices.Contains(primary, dct[0]) {
			t.Errorf("primary contains DC coefficient dct[0] = %v; Perl's diagonal walk starts at i=1 so DC must be skipped", dct[0])
		}
	})
}
