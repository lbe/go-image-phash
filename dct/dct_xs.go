package dct

import (
	"math"
)

func fct8_1d( /*inbuf []byte*/ vector []float64) {
	v0 := vector[0] + vector[7]
	v1 := vector[1] + vector[6]
	v2 := vector[2] + vector[5]
	v3 := vector[3] + vector[4]
	v4 := vector[3] - vector[4]
	v5 := vector[2] - vector[5]
	v6 := vector[1] - vector[6]
	v7 := vector[0] - vector[7]

	v8 := v0 + v3
	v9 := v1 + v2
	v10 := v1 - v2
	v11 := v0 - v3
	v12 := -v4 - v5
	v13 := (v5 + v6) * 0.707106781186547524400844
	v14 := v6 + v7

	v15 := v8 + v9
	v16 := v8 - v9
	v17 := (v10 + v11) * 0.707106781186547524400844
	v18 := (v12 + v14) * 0.382683432365089771728460

	v19 := -v12*0.541196100146196984399723 - v18
	v20 := v14*1.306562964876376527856643 - v18

	v21 := v17 + v11
	v22 := v11 - v17
	v23 := v13 + v7
	v24 := v7 - v13

	v25 := v19 + v24
	v26 := v23 + v20
	v27 := v23 - v20
	v28 := v24 - v19

	vector[0] = v15
	vector[1] = 0.509795579104157595 * v26
	vector[2] = 0.54119610014619577 * v21
	vector[3] = 0.60134488693504412 * v28
	vector[4] = 0.707106781186547 * v16
	vector[5] = 0.8999762231364133 * v25
	vector[6] = 1.30656296487637502 * v22
	vector[7] = 2.5629154477415022505 * v27
}

func fct8_2d(inbuf []float64) {
	temp := make([]float64, 64)

	for x := 0; x < 64; x += 8 {
		fct8_1d(inbuf[x : x+8])
	}

	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			temp[y*8+x] = inbuf[x*8+y]
		}
	}

	for y := 0; y < 64; y += 8 {
		fct8_1d(temp[y : y+8])
	}

	for x := 0; x < 8; x++ {
		for y := 0; y < 8; y++ {
			inbuf[y*8+x] = temp[x*8+y]
		}
	}
}

func dct_1d(inbuf []float64, size int) {
	temp := make([]float64, size)
	factor := math.Pi / float64(size)

	for i := 0; i < size; i++ {
		sum := 0.0
		mult := float64(i) * factor
		for j := 0; j < size; j++ {
			sum += inbuf[j] * math.Cos((float64(j)+0.5)*mult)
		}
		temp[i] = sum
	}

	for i := 0; i < size; i++ {
		inbuf[i] = temp[i]
	}
}

func idct_1d(inbuf []float64, size int) {
	temp := make([]float64, size)
	factor := math.Pi / float64(size)
	scale := 2.0 / float64(size)

	for i := 0; i < size; i++ {
		sum := inbuf[0] / 2.0
		mult := (float64(i) + 0.5) * factor
		for j := 1; j < size; j++ {
			sum += inbuf[j] * math.Cos(float64(j)*mult)
		}
		temp[i] = sum
	}

	for i := 0; i < size; i++ {
		inbuf[i] = temp[i] * scale
	}
}

func dct_coef(size int, coef [][]float64) {
	factor := math.Pi / float64(size)

	for i := 0; i < size; i++ {
		mult := float64(i) * factor
		for j := 0; j < size; j++ {
			coef[j][i] = math.Cos((float64(j) + 0.5) * mult)
		}
	}
}

func idct_coef(size int, coef [][]float64) {
	factor := math.Pi / float64(size)

	for i := 0; i < size; i++ {
		mult := (float64(i) + 0.5) * factor
		for j := 0; j < size; j++ {
			coef[j][i] = math.Cos(float64(j) * mult)
		}
	}
}

func dct_2d(inbuf []float64, size int) {
	temp := make([]float64, size*size)
	coef := make([][]float64, size)
	for i := range coef {
		coef[i] = make([]float64, size)
	}

	dct_coef(size, coef)

	for x := 0; x < size; x++ {
		for i := 0; i < size; i++ {
			sum := 0.0
			y := x * size
			for j := 0; j < size; j++ {
				sum += inbuf[y+j] * coef[j][i]
			}
			temp[y+i] = sum
		}
	}

	for y := 0; y < size; y++ {
		for i := 0; i < size; i++ {
			sum := 0.0
			for j := 0; j < size; j++ {
				sum += temp[j*size+y] * coef[j][i]
			}
			inbuf[i*size+y] = sum
		}
	}
}

func idct_2d(inbuf []float64, size int) {
	coef := make([][]float64, size)
	for i := range coef {
		coef[i] = make([]float64, size)
	}
	temp := make([]float64, size*size)
	scale := 2.0 / float64(size)

	idct_coef(size, coef)

	for x := 0; x < size; x++ {
		for i := 0; i < size; i++ {
			sum := inbuf[x*size] / 2.0
			y := x * size
			for j := 1; j < size; j++ {
				sum += inbuf[y+j] * coef[j][i]
			}
			temp[y+i] = sum * scale
		}
	}

	for y := 0; y < size; y++ {
		for i := 0; i < size; i++ {
			sum := temp[y] / 2.0
			for j := 1; j < size; j++ {
				sum += temp[j*size+y] * coef[j][i]
			}
			inbuf[i*size+y] = sum * scale
		}
	}
}

func fast_dct_1d_precalc(inbuf []float64, size int, coef []float64) {
	temp := make([]float64, size)

	transform_recursive(inbuf, temp, size, coef)
}

func fast_dct_coef(size int, coef []float64) {
	for i := 1; i <= size/2; i *= 2 {
		factor := math.Pi / float64(i*2)
		for j := 0; j < i; j++ {
			coef[i+j] = math.Cos((float64(j)+0.5)*factor) * 2
		}
	}
}

func fast_dct_1d(inbuf []float64, size int) {
	coef := make([]float64, size)

	fast_dct_coef(size, coef)

	fast_dct_1d_precalc(inbuf, size, coef)
}

func fast_dct_2d(inbuf []float64, size int) {
	coef := make([]float64, size)
	temp := make([]float64, size*size)

	fast_dct_coef(size, coef)

	for x := 0; x < size*size; x += size {
		fast_dct_1d_precalc(inbuf[x:x+size], size, coef)
	}

	for x := 0; x < size; x++ {
		k := x * size
		for y := 0; y < size; y++ {
			temp[y*size+x] = inbuf[k]
			k++
		}
	}

	for y := 0; y < size*size; y += size {
		fast_dct_1d_precalc(temp[y:y+size], size, coef)
	}

	for x := 0; x < size; x++ {
		k := x * size
		for y := 0; y < size; y++ {
			inbuf[y*size+x] = temp[k]
			k++
		}
	}
}

func transform_recursive(inbuf, temp []float64, size int, coef []float64) {
	if size == 1 {
		return
	}

	half := size / 2

	for i := 0; i < half; i++ {
		x := inbuf[i]
		y := inbuf[size-1-i]
		temp[i] = x + y
		temp[i+half] = (x - y) / coef[half+i]
	}

	transform_recursive(temp, inbuf, half, coef)
	transform_recursive(temp[half:], inbuf, half, coef)

	j := 0
	for i := 0; i < half-1; i++ {
		inbuf[j] = temp[i]
		j++
		inbuf[j] = temp[i+half] + temp[i+half+1]
		j++
	}
	inbuf[size-2] = temp[half-1]
	inbuf[size-1] = temp[size-1]
}
