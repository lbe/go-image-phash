// Copyright 2017 The goimagehash Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transforms

import (
	"math"
	"sync"
)

// DCT1D function returns result of DCT-II.
// DCT type II, unscaled. Algorithm by Byeong Gi Lee, 1984.
func DCT1D(input []float64) []float64 {
	temp := make([]float64, len(input))
	forwardTransform(input, temp, len(input))
	return input
}

func forwardTransform(input, temp []float64, Len int) {
	if Len == 1 {
		return
	}

	halfLen := Len / 2
	for i := 0; i < halfLen; i++ {
		x, y := input[i], input[Len-1-i]
		temp[i] = x + y
		temp[i+halfLen] = (x - y) / (math.Cos((float64(i)+0.5)*math.Pi/float64(Len)) * 2)
	}
	forwardTransform(temp, input, halfLen)
	forwardTransform(temp[halfLen:], input, halfLen)
	for i := 0; i < halfLen-1; i++ {
		input[i*2+0] = temp[i]
		input[i*2+1] = temp[i+halfLen] + temp[i+halfLen+1]
	}
	input[Len-2], input[Len-1] = temp[halfLen-1], temp[Len-1]
}

// DCT2D function returns a  result of DCT2D by using the separable property.
func DCT2D(input [][]float64, w int, h int) [][]float64 {
	output := make([][]float64, h)
	for i := range output {
		output[i] = make([]float64, w)
	}

	wg := new(sync.WaitGroup)
	for i := 0; i < h; i++ {
		wg.Add(1)
		go func(i int) {
			cols := DCT1D(input[i])
			output[i] = cols
			wg.Done()
		}(i)
	}

	wg.Wait()
	for i := 0; i < w; i++ {
		wg.Add(1)
		in := make([]float64, h)
		go func(i int) {
			for j := 0; j < h; j++ {
				in[j] = output[j][i]
			}
			rows := DCT1D(in)
			for j := 0; j < len(rows); j++ {
				output[j][i] = rows[j]
			}
			wg.Done()
		}(i)
	}

	wg.Wait()
	return output
}

// DCT2DFast64 function returns a result of DCT2D by using the separable property.
// Fast uses static DCT tables for improved performance. Returns flattened pixels.
// Added by lbe 2024-07-26
func DCT2DFast32(input *[]float64) (flattens [64]float64) {
	if len(*input) != 64*64 {
		panic("incorrect input size, wanted 64x64.")
	}

	for i := 0; i < 32; i++ { // height
		forwardDCT32((*input)[i*32 : (i*32)+32])
	}

	var row [32]float64
	for i := 0; i < 8; i++ { // width
		for j := 0; j < 32; j++ {
			row[j] = (*input)[32*j+i]
		}
		forwardDCT32(row[:])
		for j := 0; j < 8; j++ {
			flattens[8*j+i] = row[j]
		}
	}
	return
}

// DCT2DFast64 function returns a result of DCT2D by using the separable property.
// Fast uses static DCT tables for improved performance. Returns flattened pixels.
func DCT2DFast64(input *[]float64) (flattens [64 * 64]float64) {
	if len(*input) != 64*64 {
		panic("incorrect input size, wanted 64x64.")
	}

	for i := 0; i < 64; i++ { // height
		forwardDCT64((*input)[i*64 : (i*64)+64])
	}

	var row [64]float64
	for i := 0; i < 8; i++ { // width
		for j := 0; j < 64; j++ {
			row[j] = (*input)[64*j+i]
		}
		forwardDCT64(row[:])
		for j := 0; j < 64; j++ {
			flattens[8*j+i] = row[j]
		}
	}
	return
}

// DCT2DFast256 function returns a result of DCT2D by using the separable property.
// DCT type II, unscaled. Algorithm by Byeong Gi Lee, 1984.
// Fast uses static DCT tables for improved performance. Returns flattened pixels.
func DCT2DFast256(input *[]float64) (flattens [256]float64) {
	if len(*input) != 256*256 {
		panic("incorrect input size, wanted 256x256.")
	}
	for i := 0; i < 256; i++ { // height
		forwardDCT256((*input)[i*256 : 256*i+256])
	}

	var row [256]float64
	for i := 0; i < 16; i++ { // width
		for j := 0; j < 256; j++ {
			row[j] = (*input)[256*j+i]
		}
		forwardDCT256(row[:])
		for j := 0; j < 256; j++ {
			flattens[16*j+i] = row[j]
		}
	}
	return flattens
}

func Naive_perl_dct1d(vector []float64) []float64 {
	factor := math.Pi / float64(len(vector))
	result := make([]float64, len(vector))
	var sum float64

	for i := 0; i < len(vector); i++ {
		sum = 0
		for j := 0; j < len(vector); j++ {
			sum += vector[j] * math.Cos((float64(j)+0.5)*float64(i)*factor)
		}
		result[i] = sum
	}
	return result
}

func Naive_perl_dct2d(vector [][]float64) [][]float64 {
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
