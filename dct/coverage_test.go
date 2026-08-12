package dct

import (
	"errors"
	"math"
	"strconv"
	"testing"
)

func TestDCTRejectsInvalidInput(t *testing.T) {
	cases := []struct {
		name  string
		input [][]float64
		want  error
	}{
		{"empty outer", [][]float64{}, ErrInvalidInput},
		{"empty inner", [][]float64{{}}, ErrInvalidInput},
		{"non-square 2d", [][]float64{{1, 2, 3}, {4, 5, 6}}, ErrInvalidSize},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DCT(tc.input)
			if !errors.Is(err, tc.want) {
				t.Fatalf("DCT() error = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestDCT2DInfersSizeFromLength(t *testing.T) {
	input := make([]float64, 9)
	for i := range input {
		input[i] = float64(i)
	}
	var out []float64
	DCT_2D(&input, 0, &out)
	if len(out) != 9 {
		t.Fatalf("DCT_2D with sz=0 produced len %d, want 9", len(out))
	}
}

func TestIDCT2DInfersSizeFromLength(t *testing.T) {
	input := make([]float64, 16)
	for i := range input {
		input[i] = float64(i)
	}
	out := IDCT_2D(input, 0)
	if len(out) != 16 {
		t.Fatalf("IDCT_2D with sz=0 produced len %d, want 16", len(out))
	}
}

func TestDCT2DFastNMatchesDCT2DForPowerOfTwoSizes(t *testing.T) {
	for _, n := range []int{16, 32, 64, 128, 256} {
		n := n
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			input := make([]float64, n*n)
			for i := range input {
				input[i] = math.Sin(float64(i))
			}
			want := make([]float64, n*n)
			inCopy := append([]float64(nil), input...)
			DCT_2D(&inCopy, n, &want)

			got := append([]float64(nil), input...)
			DCT2DFastN(n, &got, &got)
			assertClose(t, want, got, 1e-6)
		})
	}
}

func TestDCT2DFastNBigMatchesDCT2DFor512(t *testing.T) {
	const n = 512
	input := make([]float64, n*n)
	for i := range input {
		input[i] = math.Sin(float64(i) * 0.01)
	}
	want := make([]float64, n*n)
	inCopy := append([]float64(nil), input...)
	DCT_2D(&inCopy, n, &want)

	got := append([]float64(nil), input...)
	DCT2DFastNBig(n, &got, &got)
	assertClose(t, want, got, 1e-4)
}

func TestStaticTransformFunctionsMatchTransformDCTN(t *testing.T) {
	for _, n := range []int{8, 16, 32, 64, 128, 256} {
		n := n
		t.Run(strconv.Itoa(n), func(t *testing.T) {
			input := make([]float64, n)
			for i := range input {
				input[i] = float64(i) * 0.3
			}
			want := append([]float64(nil), input...)
			transformDCTN(n, want)

			got := append([]float64(nil), input...)
			callStaticTransform(t, n, got)
			assertClose(t, want, got, 1e-9)
		})
	}
}

func TestTransformDCTNBigUsesStaticChainBelow512(t *testing.T) {
	input := make([]float64, 256)
	for i := range input {
		input[i] = math.Cos(float64(i))
	}
	want := append([]float64(nil), input...)
	transformDCT256(want)

	got := append([]float64(nil), input...)
	transformDCTNBig(256, got)
	assertClose(t, want, got, 1e-9)
}

func callStaticTransform(t *testing.T, n int, v []float64) {
	t.Helper()
	switch n {
	case 8:
		transformDCT8(v)
	case 16:
		transformDCT16(v)
	case 32:
		transformDCT32(v)
	case 64:
		transformDCT64(v)
	case 128:
		transformDCT128(v)
	case 256:
		transformDCT256(v)
	default:
		t.Fatalf("unsupported size %d", n)
	}
}

func assertClose(t *testing.T, want, got []float64, eps float64) {
	t.Helper()
	if len(want) != len(got) {
		t.Fatalf("length mismatch: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if math.Abs(want[i]-got[i]) > eps {
			t.Fatalf("index %d: got %v, want %v (eps %g)", i, got[i], want[i], eps)
		}
	}
}
