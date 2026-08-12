package phash

import (
	"errors"
	"fmt"
)

// Distance returns the Hamming distance between two hashes, matching
// Image::PHash::diff (Perl t/util.t behavior, including hashes wider than 64
// bits). It returns a clear error when the hashes have unequal bit lengths or
// contain characters other than '0' and '1'.
func Distance(a, b Hash) (int, error) {
	if len(a.bits) != len(b.bits) {
		return 0, fmt.Errorf("phash: cannot compare hashes of unequal bit lengths (%d vs %d)", len(a.bits), len(b.bits))
	}
	if !validBits(a.bits) || !validBits(b.bits) {
		return 0, errInvalidBits
	}
	dist := 0
	for i := range a.bits {
		if a.bits[i] != b.bits[i] {
			dist++
		}
	}
	return dist, nil
}

// errInvalidBits reports a hash containing characters other than '0' and '1'.
var errInvalidBits = errors.New("phash: hashes may contain only '0' and '1' bits")

// validBits reports whether s consists solely of '0' and '1' characters.
func validBits(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '0' && s[i] != '1' {
			return false
		}
	}
	return true
}
