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

// Fast uses static DCT tables for improved performance. Returns flattened pixels.
func DCT2DFast8(input []float64) (flattens [8 * 8]float64) {
	if len(input) != 8*8 {
		panic("incorrect input size, wanted 8x8.")
	}

	for i := 0; i < 8; i++ { // height
		transformDCT8((input)[i*8 : (i*8)+8])
	}

	var row [8]float64
	for i := 0; i < 8; i++ { // width
		for j := 0; j < 8; j++ {
			row[j] = (input)[8*j+i]
		}
		transformDCT8(row[:])
		for j := 0; j < 8; j++ {
			flattens[8*j+i] = row[j]
		}
	}
	return
}

// Fast uses static DCT tables for improved performance. Returns flattened pixels.
func DCT2DFast16(input []float64) (flattens [16 * 16]float64) {
	if len(input) != 16*16 {
		panic("incorrect input size, wanted 16x16.")
	}

	for i := 0; i < 16; i++ { // height
		transformDCT16((input)[i*16 : (i*16)+16])
	}

	var row [16]float64
	for i := 0; i < 16; i++ { // width
		for j := 0; j < 16; j++ {
			row[j] = (input)[16*j+i]
		}
		transformDCT16(row[:])
		for j := 0; j < 16; j++ {
			flattens[16*j+i] = row[j]
		}
	}
	return
}

// Fast uses static DCT tables for improved performance. Returns flattened pixels.
func DCT2DFast32(input []float64) (flattens [32 * 32]float64) {
	if len(input) != 32*32 {
		panic("incorrect input size, wanted 32x32.")
	}

	for i := 0; i < 32; i++ { // height
		transformDCT32((input)[i*32 : (i*32)+32])
	}

	var row [32]float64
	for i := 0; i < 32; i++ { // width
		for j := 0; j < 32; j++ {
			row[j] = (input)[32*j+i]
		}
		transformDCT32(row[:])
		for j := 0; j < 32; j++ {
			flattens[32*j+i] = row[j]
		}
	}
	return
}

// Fast uses static DCT tables for improved performance. Returns flattened pixels.
func DCT2DFast64(input []float64) (flattens [4096]float64) {
	if len(input) != 64*64 {
		panic("incorrect input size, wanted 64x64.")
	}

	for i := 0; i < 64; i++ { // height
		transformDCT64((input)[i*64 : (i*64)+64])
	}

	var row [64]float64
	for i := 0; i < 64; i++ { // width
		for j := 0; j < 64; j++ {
			row[j] = (input)[64*j+i]
		}
		transformDCT64(row[:])
		for j := 0; j < 64; j++ {
			flattens[64*j+i] = row[j]
		}
	}
	return
}

// Fast uses static DCT tables for improved performance. Returns flattened pixels.
func DCT2DFast128(input []float64) (flattens [128 * 128]float64) {
	if len(input) != 128*128 {
		panic("incorrect input size, wanted 128x128.")
	}

	for i := 0; i < 128; i++ { // height
		transformDCT128((input)[i*128 : (i*128)+128])
	}

	var row [128]float64
	for i := 0; i < 128; i++ { // width
		for j := 0; j < 128; j++ {
			row[j] = (input)[128*j+i]
		}
		transformDCT128(row[:])
		for j := 0; j < 128; j++ {
			flattens[128*j+i] = row[j]
		}
	}
	return
}

// Fast uses static DCT tables for improved performance. Returns flattened pixels.
func DCT2DFast256(input []float64) (flattens [256 * 256]float64) {
	if len(input) != 256*256 {
		panic("incorrect input size, wanted 256x256.")
	}

	for i := 0; i < 256; i++ { // height
		transformDCT256((input)[i*256 : (i*256)+256])
	}

	var row [256]float64
	for i := 0; i < 256; i++ { // width
		for j := 0; j < 256; j++ {
			row[j] = (input)[256*j+i]
		}
		transformDCT256(row[:])
		for j := 0; j < 256; j++ {
			flattens[256*j+i] = row[j]
		}
	}
	return
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
