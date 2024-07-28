package dct

import (
	"errors"
	"math"
	"slices"
)

var (
	ErrInvalidInput = errors.New("expect array of array(s)")
	ErrInvalidSize  = errors.New("expect 1d or NxN 2d arrays")
)

func DCT(vector [][]float64) ([][]float64, error) {
	if len(vector) == 0 || len(vector[0]) == 0 {
		return nil, ErrInvalidInput
	}

	dim := len(vector)
	sz := len(vector[0])
	if dim != 1 && dim != sz {
		return nil, ErrInvalidSize
	}

	pack := flatten(vector)

	if dim > 1 {
		if sz == 8 {
			fct8_2d(pack)
		} else if (sz & (sz - 1)) == 0 {
			fast_dct_2d(pack, sz)
		} else {
			dct_2d(pack, sz)
		}
	} else {
		if sz == 8 {
			fct8_1d(pack)
		} else if (sz & (sz - 1)) == 0 {
			fast_dct_1d(pack, sz)
		} else {
			dct_1d(pack, sz)
		}
	}

	result := make([][]float64, dim)
	for i := 0; i < dim; i++ {
		result[i] = make([]float64, sz)
	}

	for i := 0; i < dim; i++ {
		for j := 0; j < sz; j++ {
			result[i][j] = pack[(i*sz + j)]
		}
	}
	return result, nil
}

func DCT_1D(input []float64, sz int) []float64 {
	result := slices.Clone(input)

	if sz == 8 {
		fct8_1d(result)
		return result
	}

	if (sz & (sz - 1)) == 0 {
		fast_dct_1d(result, sz)
		return result
	}

	dct_1d(result, sz)
	return result
}

func IDCT_1D(input []float64, sz int) []float64 {
	result := slices.Clone(input)

	idct_1d(result, sz)

	return result
}

func DCT_2D(input []float64, sz int) []float64 {
	if sz == 0 {
		sz = int(math.Sqrt(float64(len(input))))
	}

	result := slices.Clone(input)
	if sz == 8 {
		fct8_2d(result) // Arai, Agui, Nakajima
		return result
	}

	if (sz & (sz - 1)) == 0 { // power of 2
		fast_dct_2d(result, sz) // Lee
		return result
	}

	dct_2d(result, sz)
	return result
}

func IDCT_2D(input []float64, sz int) []float64 {
	if sz == 0 {
		sz = int(math.Sqrt(float64(len(input))))
	}

	result := slices.Clone(input)
	idct_2d(result, sz)

	//result := make([]float64, sz*sz)
	//for i := 0; i < sz*sz; i++ {
	//	result[i] = input[i*8]
	//}
	return result
}

// flatten [][] to [] to match convention ported from perl Math::DCT
func flatten(in [][]float64) []float64 {
	h := len(in)
	w := len(in[0])
	out := make([]float64, h*w)
	for i, row := range in {
		for j, col := range row {
			out[i*h+j] = col
		}
	}
	return out
}
