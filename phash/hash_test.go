package phash_test

import (
	"strings"
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// TestHashFromBitsExposesHexBitLengthAndZeroState verifies that a Hash
// constructed from a bit sequence reports the same hex as BitsToHex (matched
// against a hardcoded golden, including a 35-bit length not divisible by 4),
// that BitLen returns the original bit count (retained alongside the hex
// rather than derived from it), and that IsZero is true only for the
// empty/all-zero hash.
func TestHashFromBitsExposesHexBitLengthAndZeroState(t *testing.T) {
	cases := []struct {
		name     string
		bits     string
		wantHex  string
		wantLen  int
		wantZero bool
	}{
		{
			name:     "64-bit all ones",
			bits:     "1111111111111111111111111111111111111111111111111111111111111111",
			wantHex:  "FFFFFFFFFFFFFFFF",
			wantLen:  64,
			wantZero: false,
		},
		{
			name:     "35-bit length not divisible by four",
			bits:     "10101010101010101010101010101010101",
			wantHex:  "555555555",
			wantLen:  35,
			wantZero: false,
		},
		{
			name:     "all-zero bits",
			bits:     "0000000000000000000000000000000000000000000000000000000000000000",
			wantHex:  "0000000000000000",
			wantLen:  64,
			wantZero: true,
		},
		{
			name:     "empty bit sequence",
			bits:     "",
			wantHex:  "",
			wantLen:  0,
			wantZero: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := phash.NewHash(tc.bits)

			if got := h.Hex(); got != tc.wantHex {
				t.Errorf("Hex() = %q, want %q (golden)", got, tc.wantHex)
			}
			if got := h.Hex(); got != phash.BitsToHex(tc.bits) {
				t.Errorf("Hex() = %q, want BitsToHex(%q) = %q", got, tc.bits, phash.BitsToHex(tc.bits))
			}
			if got := h.BitLen(); got != tc.wantLen {
				t.Errorf("BitLen() = %d, want %d (original bit count)", got, tc.wantLen)
			}
			if got := h.IsZero(); got != tc.wantZero {
				t.Errorf("IsZero() = %v, want %v", got, tc.wantZero)
			}
		})
	}
}

// TestHashUint64SucceedsOnlyWhenBitLengthIsAtMost64 verifies that Uint64
// returns the numeric value of the hash whenever BitLen is 64 or fewer bits
// (including a 35-bit value whose length is not a multiple of 4 and the
// 64-bit all-ones maximum), and that a hash longer than 64 bits yields an
// error instead of a truncated or bogus value.
func TestHashUint64SucceedsOnlyWhenBitLengthIsAtMost64(t *testing.T) {
	cases := []struct {
		name    string
		bits    string
		want    uint64
		wantErr bool
	}{
		{
			name: "64-bit all ones is the maximum uint64",
			bits: strings.Repeat("1", 64),
			want: ^uint64(0),
		},
		{
			name: "35-bit value whose length is not divisible by four",
			bits: "10101010101010101010101010101010101",
			want: 22906492245,
		},
		{
			name: "single set bit",
			bits: "1",
			want: 1,
		},
		{
			name: "empty hash is zero",
			bits: "",
			want: 0,
		},
		{
			name:    "65-bit hash exceeds uint64",
			bits:    strings.Repeat("1", 65),
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := phash.NewHash(tc.bits)

			got, err := h.Uint64()
			if tc.wantErr {
				if err == nil {
					t.Errorf("Uint64() = %d, want an error for a %d-bit hash (longer than 64)", got, h.BitLen())
				}
				return
			}
			if err != nil {
				t.Fatalf("Uint64() returned an unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Uint64() = %d, want %d", got, tc.want)
			}
		})
	}
}
