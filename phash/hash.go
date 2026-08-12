// Package phash implements perceptual hashing with algorithmic parity to
// Image::PHash 0.3.
package phash

import (
	"errors"
	"strconv"
	"strings"
)

// Hash is a perceptual hash constructed from a binary bit sequence. The
// original bit string is retained so BitLen is exact even when the bit count
// is not a multiple of 4, a length that hex digits alone cannot convey.
type Hash struct {
	bits string
}

// NewHash constructs a Hash from a binary bit string of '0' and '1'
// characters.
func NewHash(bits string) Hash { return Hash{bits: bits} }

// Hex returns the uppercase hex form of the hash, matching BitsToHex.
func (h Hash) Hex() string { return BitsToHex(h.bits) }

// BitLen returns the number of bits in the hash.
func (h Hash) BitLen() int { return len(h.bits) }

// IsZero reports whether the hash has no set bits.
func (h Hash) IsZero() bool {
	return strings.IndexByte(h.bits, '1') == -1
}

// Uint64 returns the numeric value of the hash when its bit length is at most
// 64, and an error for longer hashes whose value does not fit in a uint64.
func (h Hash) Uint64() (uint64, error) {
	if len(h.bits) > uint64Bits {
		return 0, errHashTooLong
	}
	if h.bits == "" {
		return 0, nil
	}
	return strconv.ParseUint(h.bits, 2, uint64Bits)
}

// errHashTooLong reports a hash wider than 64 bits, whose value cannot fit in
// a uint64.
var errHashTooLong = errors.New("phash: hash longer than 64 bits does not fit in a uint64")

// uint64Bits is the number of bits in a uint64.
const uint64Bits = 64
