package dct

import (
	"strings"
	"testing"
)

func TestDCT2DFastNPanicsOnInvalidSize(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("DCT2DFastN with invalid size must panic")
		}
		if msg, ok := r.(string); !ok || !strings.Contains(msg, "not a power of 2") {
			t.Fatalf("panic = %v, want power-of-2 message", r)
		}
	}()
	var in, out []float64
	DCT2DFastN(3, &in, &out)
}

func TestDCT2DFastNBigPanicsOnInvalidSize(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("DCT2DFastNBig with invalid size must panic")
		}
	}()
	var in, out []float64
	DCT2DFastNBig(3, &in, &out)
}

func TestTransformDCTNPanicsWhenTableMissing(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("transformDCTN with unsupported size must panic")
		}
	}()
	v := make([]float64, 3)
	transformDCTN(3, v)
}
