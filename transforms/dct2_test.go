// Copyright 2017 The goimagehash Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package transforms

import (
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
		exp[i] = Naive_perl_dct1d(ary[i])
		exp2d[i] = Naive_perl_dct2d(ary2d[i])
	}
}

func TestDCT1D(t *testing.T) {
	for _, tt := range []struct {
		input  []float64
		output []float64
	}{
		//{[]float64{1.0, 1.0, 1.0, 1.0}, []float64{4.0, 0, 0, 0}},
		//{[]float64{1.0, 2.0, 3.0, 4.0}, []float64{10.0, -3.15432202989895, 0.0, -0.224170764583983}},
		/*{
			[]float64{0.3181653197002592, 0.39066343796185155, 0.16102608753078032},
			[]float64{0.797356726931299, 0.275539249463622, -0.32010874738091},
		},*/
		{ary[3], exp[3]},
		{ary[4], exp[4]},
		{ary[8], exp[8]},
		{ary[32], exp[32]},
		{ary[64], exp[64]},
	} {

		out := DCT1D(tt.input)
		pass := true

		if len(tt.output) != len(out) {
			t.Errorf("DCT1D(%v) is expected %v but got %v.", tt.input, tt.output, out)
		}

		for i := range out {
			if (out[i]-tt.output[i]) > EPSILON || (tt.output[i]-out[i]) > EPSILON {
				pass = false
			}
		}

		if !pass || len(tt.output) != len(out) {
			t.Errorf("DCT1D(%v) is expected %v but got %v.", tt.input, tt.output, out)
		}
	}
}

func TestDCT2D(t *testing.T) {
	for _, tt := range []struct {
		input  [][]float64
		output [][]float64
		w      int
		h      int
	}{
		{
			[][]float64{
				{1.0, 2.0, 3.0, 4.0},
				{5.0, 6.0, 7.0, 8.0},
				{9.0, 10.0, 11.0, 12.0},
				{13.0, 14.0, 15.0, 16.0},
			},
			[][]float64{
				{136.0, -12.6172881195958, 0.0, -0.8966830583359305},
				{-50.4691524783832, 0.0, 0.0, 0.0},
				{0.0, 0.0, 0.0, 0.0},
				{-3.586732233343722, 0.0, 0.0, 0.0},
			},
			4, 4,
		},
		{
			[][]float64{
				{1.0, 2.0},
				{3.0, 4.0},
			},
			[][]float64{
				{10.0, -1.41421356237309},
				{-2.82842712474619, 0},
			},
			2,
			2,
		},
	} {
		out := DCT2D(tt.input, tt.w, tt.h)
		pass := true

		for i := 0; i < tt.h; i++ {
			for j := 0; j < tt.w; j++ {
				if (out[i][j]-tt.output[i][j]) > EPSILON || (tt.output[i][j]-out[i][j]) > EPSILON {
					pass = false
				}
			}
		}

		if !pass {
			t.Errorf("DCT2D(%v) is expected %v but got %v.", tt.input, tt.output, out)
		}
	}
}

func init() {
	createTestData()
}
