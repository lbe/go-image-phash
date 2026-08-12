package phash

import (
	"testing"
)

func TestSelectGeometryRejectsMalformedGrid(t *testing.T) {
	dct := make([]float64, 32*32)
	primary, extra := selectGeometry(dct, "not-a-grid", false, "average", 32)
	if primary != nil || extra != nil {
		t.Fatalf("selectGeometry(bad geometry) = (%v, %v), want (nil, nil)", primary, extra)
	}
}

func TestSelectBitCountWhenWalkIsShorterThanRequestedBits(t *testing.T) {
	// resize 4 yields fewer than 10 diagonal-walk coefficients.
	dct := make([]float64, 16)
	for i := range dct {
		dct[i] = float64(i)
	}
	primary, extra := selectBitCount(dct, 10, "median", 4)
	if len(primary) == 10 {
		t.Fatalf("primary has %d coeffs, want fewer than 10 when resize is 4", len(primary))
	}
	if extra != nil {
		t.Fatalf("extra = %v, want nil when walk cannot supply K primary coeffs", extra)
	}
}

func TestNormalizeHashOptionsAppliesResizeDefault(t *testing.T) {
	opts, resize, err := normalizeHashOptions(Options{}, 0)
	if err != nil {
		t.Fatalf("normalizeHashOptions with resize 0: %v", err)
	}
	if resize != defaultResize {
		t.Fatalf("resize = %d, want default %d", resize, defaultResize)
	}
	if opts.Geometry != defaultGeometry {
		t.Fatalf("geometry = %q, want %q", opts.Geometry, defaultGeometry)
	}
}

func TestNormalizeHashOptionsRejectsInvalidGeometry(t *testing.T) {
	_, _, err := normalizeHashOptions(Options{Geometry: "bad"}, 32)
	if err == nil {
		t.Fatal("normalizeHashOptions with bad geometry must error")
	}
}

func TestParseBitCountRejectsNonNumericGeometry(t *testing.T) {
	if _, ok := parseBitCount("8x8"); ok {
		t.Fatal("parseBitCount must not accept NxN strings")
	}
}

func TestParseGridGeometryRejectsMalformedStrings(t *testing.T) {
	for _, g := range []string{"", "x8", "8x", "8x8x8"} {
		if _, _, ok := parseGridGeometry(g); ok {
			t.Fatalf("parseGridGeometry(%q) = ok, want false", g)
		}
	}
}
