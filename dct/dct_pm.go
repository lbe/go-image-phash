package dct

import (
	"errors"
	"fmt"
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
		switch {
		case sz == 8:
			FDCT8_2D(pack)
		case (sz & (sz - 1)) == 0:
			fast_dct_2d(pack, sz)
		default:
			dct_2d(pack, sz)
		}
	} else {
		switch {
		case sz == 8:
			FDCT8_1D(pack)
		case (sz & (sz - 1)) == 0:
			fast_dct_1d(pack, sz)
		default:
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
		FDCT8_1D(result)
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

func DCT_2D(input *[]float64, sz int, result *[]float64) {
	if sz == 0 {
		sz = int(math.Sqrt(float64(len(*input))))
	}

	switch {
	case sz == 8:
		{
			*result = *input
			FDCT8_2D(*result) // Arai, Agui, Nakajima
		}
	case sz < 512 && (sz&(sz-1)) == 0: // power of 2
		DCT2DFastN(sz, input, result)
	case (sz & (sz - 1)) == 0: // power of 2
		DCT2DFastNBig(sz, input, result)
	default:
		{
			*result = *input
			dct_2d(*result, sz)
		}
	}
}

func IDCT_2D(input []float64, sz int) []float64 {
	if sz == 0 {
		sz = int(math.Sqrt(float64(len(input))))
	}

	result := slices.Clone(input)
	idct_2d(result, sz)

	return result
}

// Fast uses static DCT tables for improved performance. Returns flattened pixels.
func DCT2DFastN(n int, input *[]float64, flattens *[]float64) {
	if (n < 4) && (n&(n-1) != 0) {
		panic(fmt.Sprintf("DCT2DFastN = %d is not a power of 2 or is < 4", n))
	}

	for i := 0; i < n; i++ { // height
		transformDCTN(n, (*input)[i*n:(i*n)+n])
	}

	row := make([]float64, n)
	for i := 0; i < n; i++ { // width
		for j := 0; j < n; j++ {
			row[j] = (*input)[n*j+i]
		}
		transformDCTN(n, row)
		for j := 0; j < n; j++ {
			(*flattens)[n*j+i] = row[j]
		}
	}
}

// Fast uses static DCT tables for improved performance. Returns flattened pixels.
func DCT2DFastNBig(n int, input *[]float64, flattens *[]float64) {
	if (n < 4) && (n&(n-1) != 0) {
		panic(fmt.Sprintf("transformDCTN n = %d is not a power of 2 or is < 4", n))
	}

	for i := 0; i < n; i++ { // height
		transformDCTNBig(n, (*input)[i*n:(i*n)+n])
	}

	row := make([]float64, n)
	for i := 0; i < n; i++ { // width
		for j := 0; j < n; j++ {
			row[j] = (*input)[n*j+i]
		}
		transformDCTNBig(n, row)
		for j := 0; j < n; j++ {
			(*flattens)[n*j+i] = row[j]
		}
	}
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
