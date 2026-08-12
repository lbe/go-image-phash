// Package phash implements perceptual hashing with algorithmic parity to
// Image::PHash 0.3.
package phash

const hexDigits = "0123456789ABCDEF"

// BitsToHex converts a binary bit string (characters '0' and '1') to an
// uppercase hexadecimal string, matching Image::PHash::b2h. Every 4-bit chunk
// becomes one hex digit; the leading chunk may be shorter when the input
// length is not divisible by 4.
func BitsToHex(bits string) string {
	out := make([]byte, 0, len(bits)/4+1)
	head := len(bits) % 4
	if head > 0 {
		out = append(out, hexDigits[bitsToInt(bits[:head])])
	}
	for pos := head; pos < len(bits); pos += 4 {
		out = append(out, hexDigits[bitsToInt(bits[pos:pos+4])])
	}
	return string(out)
}

func bitsToInt(bits string) int {
	val := 0
	for _, c := range bits {
		val = val<<1 | int(c-'0')
	}
	return val
}
