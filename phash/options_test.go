package phash_test

import (
	"strings"
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// TestValidateOptionsRejectsIllegalGeometryMethodAndMirrorCombinations
// verifies that the standalone Options validator, aligned with Image::PHash
// 0.3's _validate_options, rejects non-square or malformed geometry shapes,
// unknown methods, NxN geometries larger than the resize value (with a
// non-positive resize defaulting to 32), and the mutually exclusive
// mirror/mirrorproof pair — and that zero-valued Options are normalized to
// the Image::PHash defaults (geometry '8x8', method 'average') with no error.
func TestValidateOptionsRejectsIllegalGeometryMethodAndMirrorCombinations(t *testing.T) {
	cases := []struct {
		name       string
		opts       phash.Options
		resize     int
		wantErr    bool
		errPart    string // expected substring of the error when wantErr
		wantGeo    string // expected normalized geometry for valid cases
		wantMethod string // expected normalized method for valid cases
	}{
		{
			name:    "non-square NxN geometry is rejected",
			opts:    phash.Options{Geometry: "8x4"},
			resize:  32,
			wantErr: true,
			errPart: "geometry",
		},
		{
			name:    "garbage geometry is rejected",
			opts:    phash.Options{Geometry: "abc"},
			resize:  32,
			wantErr: true,
			errPart: "geometry",
		},
		{
			name:    "NxN with an empty side is rejected",
			opts:    phash.Options{Geometry: "8x"},
			resize:  32,
			wantErr: true,
			errPart: "geometry",
		},
		{
			name:    "unknown method is rejected",
			opts:    phash.Options{Method: "bogus"},
			resize:  32,
			wantErr: true,
			errPart: "method",
		},
		{
			name:    "NxN geometry larger than the resize is rejected",
			opts:    phash.Options{Geometry: "33x33"},
			resize:  32,
			wantErr: true,
			errPart: "resize",
		},
		{
			name:    "NxN geometry larger than the default resize 32 is rejected",
			opts:    phash.Options{Geometry: "33x33"},
			resize:  0,
			wantErr: true,
			errPart: "resize",
		},
		{
			name:    "mirror and mirrorproof together are rejected",
			opts:    phash.Options{Mirror: true, MirrorProof: true},
			resize:  32,
			wantErr: true,
			errPart: "mirror",
		},
		{
			name:       "zero-valued options apply Image::PHash defaults",
			opts:       phash.Options{},
			resize:     0,
			wantGeo:    "8x8",
			wantMethod: "average",
		},
		{
			name:       "bit-count geometry passes without a resize check",
			opts:       phash.Options{Geometry: "64"},
			resize:     0,
			wantGeo:    "64",
			wantMethod: "average",
		},
		{
			name:   "NxN geometry equal to the resize passes",
			opts:   phash.Options{Geometry: "32x32"},
			resize: 32,
		},
		{
			name:       "median method passes",
			opts:       phash.Options{Method: "median"},
			resize:     32,
			wantMethod: "median",
		},
		{
			name:       "average_x method passes",
			opts:       phash.Options{Method: "average_x"},
			resize:     32,
			wantMethod: "average_x",
		},
		{
			name:       "log method passes",
			opts:       phash.Options{Method: "log"},
			resize:     32,
			wantMethod: "log",
		},
		{
			name:       "diff method passes",
			opts:       phash.Options{Method: "diff"},
			resize:     32,
			wantMethod: "diff",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := phash.Validate(tc.opts, tc.resize)
			if tc.wantErr {
				if err == nil {
					t.Errorf("Validate(%+v, %d) = %+v, nil error; want an error mentioning %q", tc.opts, tc.resize, got, tc.errPart)
					return
				}
				if !strings.Contains(strings.ToLower(err.Error()), tc.errPart) {
					t.Errorf("Validate(%+v, %d) error = %q, want one mentioning %q", tc.opts, tc.resize, err, tc.errPart)
				}
				return
			}
			if err != nil {
				t.Errorf("Validate(%+v, %d) returned an unexpected error: %v", tc.opts, tc.resize, err)
				return
			}
			if tc.wantGeo != "" && got.Geometry != tc.wantGeo {
				t.Errorf("Validate(%+v, %d).Geometry = %q, want %q (Image::PHash default applied)", tc.opts, tc.resize, got.Geometry, tc.wantGeo)
			}
			if tc.wantMethod != "" && got.Method != tc.wantMethod {
				t.Errorf("Validate(%+v, %d).Method = %q, want %q (Image::PHash default applied)", tc.opts, tc.resize, got.Method, tc.wantMethod)
			}
		})
	}
}
