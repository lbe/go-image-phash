package dct

import (
	"math"
	"math/rand"
	"testing"
)

const (
	EPSILON float64 = 0.00000001
)

// input into tests
var (
	ary, ary2d_flat map[int][]float64
	ary2d           map[int][][]float64
)

// expect output of tests
var (
	exp   map[int][]float64
	exp2d map[int][][]float64
)

// function to popular above
func createTestData() {
	r := rand.New(rand.NewSource(99))
	ints := []int{3, 4, 8, 11, 32, 64}
	ary = make(map[int][]float64)
	ary2d = make(map[int][][]float64)
	ary2d_flat = make(map[int][]float64)
	for _, i := range ints {
		for j := 0; j < i; j++ {
			sf64 := []float64{}
			for range i {
				sf64 = append(sf64, r.Float64())
			}
			ary[i] = sf64
			ary2d[i] = append(ary2d[i], []float64(sf64))
			ary2d_flat[i] = append(ary2d_flat[i], sf64...)
		}
	}
	exp = make(map[int][]float64)
	exp2d = make(map[int][][]float64)
	for _, i := range ints {
		exp[i] = naive_dct1d(ary[i])
		exp2d[i] = naive_dct2d(ary2d[i])
	}
}

func TestDCT_1D(t *testing.T) {
	for _, tt := range []struct {
		input  []float64
		output []float64
	}{
		{ary[3], exp[3]},
		{ary[4], exp[4]},
		{ary[8], exp[8]},
		{ary[11], exp[11]},
		{ary[32], exp[32]},
		{ary[64], exp[64]},
	} {
		out := DCT_1D(tt.input, len(tt.input))
		pass := true

		if len(tt.output) != len(out) {
			t.Errorf("DCT_1D(%v) is expected %v but got %v.", tt.input, tt.output, out)
		}

		for i := range out {
			if math.Abs(out[i]-tt.output[i]) > EPSILON {
				pass = false
			}
		}

		if !pass || len(tt.output) != len(out) {
			t.Errorf("DCT_1D(%v) expected %v but got %v.", tt.input, tt.output, out)
		}
	}
}

func TestIDCT_1D(t *testing.T) {
	for _, tt := range []struct {
		input  []float64
		output []float64
	}{
		{ary[3], exp[3]},
		{ary[4], exp[4]},
		{ary[8], exp[8]},
		{ary[11], exp[11]},
		{ary[32], exp[32]},
		{ary[64], exp[64]},
	} {
		in := IDCT_1D(tt.output, len(tt.output))
		pass := true

		if len(tt.input) != len(in) {
			t.Errorf("IDCT_1D(%v) is expected %v but got %v.", tt.input, tt.output, in)
		}

		for i := range in {
			if math.Abs(in[i]-tt.input[i]) > EPSILON {
				pass = false
			}
		}

		if !pass || len(tt.input) != len(in) {
			t.Errorf("IDCT_1D(%v) expected %v but got %v.", tt.output, tt.input, in)
		}
	}
}

func TestDCT_2D(t *testing.T) {
	for _, tt := range []struct {
		input  [][]float64
		output [][]float64
		w      int
		h      int
	}{
		{ary2d[3], exp2d[3], 3, 3},
		{ary2d[4], exp2d[4], 4, 4},
		{ary2d[8], exp2d[8], 8, 8},
		{ary2d[11], exp2d[11], 11, 11},
		{ary2d[32], exp2d[32], 32, 32},
		{ary2d[64], exp2d[64], 64, 64},
	} {
		flat_in := flatten(tt.input)
		out := DCT_2D(flat_in, tt.w)
		pass := true

		for i := 0; i < tt.w; i++ {
			for j := 0; j < tt.h; j++ {
				if (out[i*tt.w+j]-tt.output[i][j]) > EPSILON || (tt.output[i][j]-out[i*tt.w+j]) > EPSILON {
					pass = false
				}
			}
		}

		if !pass {
			t.Errorf("DCT_2D(%d, %d, %v) expected %v but got %v.", tt.w, tt.h, tt.input, tt.output, out)
		}
	}
}

func TestIDCT_2D(t *testing.T) {
	for _, tt := range []struct {
		input  [][]float64
		output [][]float64
		w      int
		h      int
	}{
		{ary2d[3], exp2d[3], 3, 3},
		{ary2d[4], exp2d[4], 4, 4},
		{ary2d[8], exp2d[8], 8, 8},
		{ary2d[11], exp2d[11], 11, 11},
		{ary2d[32], exp2d[32], 32, 32},
		{ary2d[64], exp2d[64], 64, 64},
	} {
		flat_out := flatten(tt.output)
		in := IDCT_2D(flat_out, tt.w)
		pass := true

		for i := 0; i < tt.w; i++ {
			for j := 0; j < tt.h; j++ {
				if math.Abs(in[i*tt.w+j]-tt.input[i][j]) > EPSILON {
					pass = false
				}
			}
		}

		if !pass {
			t.Errorf("DCT2D(%d, %d, %v) expected %v but got %v.", tt.w, tt.h, tt.output, tt.input, in)
		}
	}
}

func TestDCT(t *testing.T) {
	for _, tt := range []struct {
		input  [][]float64
		output [][]float64
		w      int
		h      int
	}{
		{[][]float64{ary[3]}, [][]float64{exp[3]}, 3, 1},
		{[][]float64{ary[4]}, [][]float64{exp[4]}, 4, 1},
		{[][]float64{ary[8]}, [][]float64{exp[8]}, 8, 1},
		{[][]float64{ary[11]}, [][]float64{exp[11]}, 11, 1},
		{[][]float64{ary[32]}, [][]float64{exp[32]}, 32, 1},
		{[][]float64{ary[64]}, [][]float64{exp[64]}, 64, 1},
		{ary2d[3], exp2d[3], 3, 3},
		{ary2d[4], exp2d[4], 4, 4},
		{ary2d[8], exp2d[8], 8, 8},
		{ary2d[11], exp2d[11], 11, 11},
		{ary2d[32], exp2d[32], 32, 32},
		{ary2d[64], exp2d[64], 64, 64},
	} {
		out, err := DCT(tt.input)
		if err != nil {
			t.Errorf("DCT(%d, %d, %v) returned error %v", tt.w, tt.h, tt.input, err)
		}
		pass := true

		for i := 0; i < tt.h; i++ {
			for j := 0; j < tt.w; j++ {
				if math.Abs(out[i][j]-tt.output[i][j]) > EPSILON {
					pass = false
				}
			}
		}

		if !pass {
			t.Errorf("DCT(%d, %d, %v) expected %v but got %v.", tt.w, tt.h, tt.input, tt.output, out)
		}
	}
}

func init() {
	createTestData()
}

func naive_dct1d(vector []float64) []float64 {
	factor := math.Pi / float64(len(vector))
	result := make([]float64, len(vector))

	for i := 0; i < len(vector); i++ {
		sum := 0.0
		for j := 0; j < len(vector); j++ {
			sum += vector[j] * math.Cos((float64(j)+0.5)*float64(i)*factor)
		}
		result[i] = sum
	}
	return result
}

func naive_dct2d(vector [][]float64) [][]float64 {
	N := len(vector)
	factor := math.Pi / float64(N)
	temp := make([][]float64, N)
	result := make([][]float64, N)

	for x := 0; x < N; x++ {
		temp[x] = make([]float64, N)
		for i := 0; i < N; i++ {
			sum := 0.0
			for j := 0; j < N; j++ {
				sum += vector[x][j] * math.Cos((float64(j)+0.5)*float64(i)*factor)
			}
			temp[x][i] = sum
		}
	}

	for y := 0; y < N; y++ {
		result[y] = make([]float64, N)
	}

	for y := 0; y < N; y++ {
		for i := 0; i < N; i++ {
			sum := 0.0
			for j := 0; j < N; j++ {
				sum += temp[j][y] * math.Cos((float64(j)+0.5)*float64(i)*factor)
			}
			result[i][y] = sum
		}
	}
	return result
}
