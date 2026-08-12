package phash_test

import (
	"strings"
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// TestDistanceMatchesImagePHashDiff verifies that Distance returns the same
// Hamming distances as Image::PHash 0.3's diff for the two cases asserted in
// the Perl t/util.t "diff" subtest — the 4-bit pair ('A','B') and the
// >64-bit pair of 17-hex-digit strings — and that unequal-length or
// malformed hashes yield a clear error instead of a bogus distance (Go
// reports errors where Perl silently compares mismatched chunks).
func TestDistanceMatchesImagePHashDiff(t *testing.T) {
	cases := []struct {
		name    string
		a, b    string // bit strings matching the hex pair (every nibble is 4 bits)
		want    int
		wantErr bool
	}{
		{
			name: "perl util.t diff of hex A and B (4 bits)",
			a:    "1010", // hex "A"
			b:    "1011", // hex "B"
			want: 1,
		},
		{
			name: "perl util.t diff of 17-hex-digit pair (68 bits, >64)",
			a:    strings.Repeat("1010", 17),          // hex "AAAAAAAAAAAAAAAAA"
			b:    strings.Repeat("1010", 16) + "1011", // hex "AAAAAAAAAAAAAAAAB"
			want: 1,
		},
		{
			name: "identical 64-bit hashes",
			a:    strings.Repeat("1010", 16), // hex "AAAAAAAAAAAAAAAA"
			b:    strings.Repeat("1010", 16), // hex "AAAAAAAAAAAAAAAA"
			want: 0,
		},
		{
			name: "hex F vs 0 differs in all four bits",
			a:    "1111", // hex "F"
			b:    "0000", // hex "0"
			want: 4,
		},
		{
			name: "hex ABCD vs ABCC differs in one bit",
			a:    "1010101111001101", // hex "ABCD"
			b:    "1010101111001100", // hex "ABCC"
			want: 1,
		},
		{
			name:    "unequal bit lengths return an error",
			a:       "1010",     // hex "A"
			b:       "10101010", // hex "AA"
			wantErr: true,
		},
		{
			name:    "hash with non-bit characters returns an error",
			a:       "1020",
			b:       "1010",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := phash.Distance(phash.NewHash(tc.a), phash.NewHash(tc.b))
			if err != nil && !tc.wantErr {
				t.Errorf("Distance(%q, %q) returned unexpected error %v", tc.a, tc.b, err)
				return
			}
			if err == nil && tc.wantErr {
				t.Errorf("Distance(%q, %q) = %d, nil error; want an error", tc.a, tc.b, got)
				return
			}
			if got != tc.want {
				t.Errorf("Distance(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}
