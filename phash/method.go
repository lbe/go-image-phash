package phash

import (
	"math"
	"sort"
	"strings"
)

// applyAverage converts the selected coefficients into a bit string using
// Image::PHash 0.3's average threshold: the mean of primary[skip:] plus any
// extras (see thresholdMean), with each primary coefficient becoming '1'
// when it is at or above that mean and '0' otherwise (see bitsAtOrAbove).
// The first skip coefficients are excluded from the mean but still produce
// bits: for a non-reduce NxN geometry skip is 1 and primary[0] is DC. The
// average method never receives extras (Image::PHash's $x_mult is false for
// it), so extra is expected to be empty.
func applyAverage(primary []float64, skip int, extra []float64) string {
	return bitsAtOrAbove(primary, thresholdMean(primary, skip, extra))
}

// applyMedian converts the selected coefficients into a bit string using
// Image::PHash 0.3's median threshold over the combined primary and extra
// values (see medianThreshold), with each primary coefficient becoming '1'
// when it is at or above that threshold and '0' otherwise (see
// bitsAtOrAbove). Extras take part in the threshold but produce no bits. For
// a non-reduce NxN geometry skip is 1 and primary[0] is DC; the reduce and
// bit-count geometries carry skip=0.
func applyMedian(primary []float64, skip int, extra []float64) string {
	return bitsAtOrAbove(primary, medianThreshold(primary, skip, extra))
}

// applyAverageX converts the selected coefficients into a bit string using
// Image::PHash 0.3's average_x method. Perl registers average_x as
// \&_apply_average — the same subroutine as average — but selects it with
// $x_mult true, so the pHash loop feeds the threshold the same @extra lists
// as the median method. applyAverageX therefore delegates to applyAverage:
// extras take part in the mean but produce no bits. For a non-reduce NxN
// geometry skip is 1 and primary[0] is DC; the reduce and bit-count
// geometries carry skip=0.
func applyAverageX(primary []float64, skip int, extra []float64) string {
	return applyAverage(primary, skip, extra)
}

// applyLog converts the selected coefficients into a bit string using
// Image::PHash 0.3's log threshold (see logThreshold), with each primary
// coefficient becoming '1' when it is at or above that threshold and '0'
// otherwise (see bitsAtOrAbove). The first skip coefficients are excluded
// from the threshold but still produce bits: for a non-reduce NxN geometry
// skip is 1 and primary[0] is DC; the reduce and bit-count geometries carry
// skip=0. The log method never receives extras (Image::PHash's $x_mult is
// false for it), so extra is expected to be empty.
func applyLog(primary []float64, skip int, extra []float64) string {
	return bitsAtOrAbove(primary, logThreshold(primary, skip))
}

// applyDiff converts the selected coefficients into a bit string using
// Image::PHash 0.3's _apply_diff: the first coefficient becomes '1' when it
// is greater than 0 and '0' otherwise, and each later coefficient becomes '1'
// when it is greater than the coefficient before it and '0' otherwise (see
// diffBit), so the bits depend only on the consecutive ordering of the
// original primary values. Unlike the other methods, _apply_diff ignores both
// the Perl $skip flag and @extra: every primary coefficient produces a bit
// regardless of skip (the pHash loop still passes skip=1 for a non-reduce
// NxN geometry, but diff ignores it). The diff method never receives extras
// (Image::PHash's $x_mult is false for it), so extra is expected to be empty.
func applyDiff(primary []float64, skip int, extra []float64) string {
	var b strings.Builder
	b.Grow(len(primary))
	if len(primary) > 0 {
		b.WriteByte(diffBit(primary[0], 0))
	}
	for i := 1; i < len(primary); i++ {
		b.WriteByte(diffBit(primary[i], primary[i-1]))
	}
	return b.String()
}

// diffBit returns the Image::PHash diff bit for a coefficient compared
// against another value: '1' when the coefficient is greater and '0'
// otherwise.
func diffBit(v, comparedTo float64) byte {
	if v > comparedTo {
		return '1'
	}
	return '0'
}

// logThreshold returns Image::PHash 0.3's log threshold (see
// _apply_log_average): the mean of sign-preserving (1/exp)-powered
// magnitudes of primary[skip:], raised back to the exp power, where exp is
// 1 + len(primary)^(1/6) computed over the full primary list. Each
// coefficient keeps its sign through the powering so the mean can cancel
// toward zero; the final threshold restores the sign of the mean. The first
// skip coefficients take no part in the threshold, matching Image::PHash's
// $skip flag; they still produce bits in the caller's bit string.
func logThreshold(primary []float64, skip int) float64 {
	exp := 1 + math.Pow(float64(len(primary)), 1.0/6.0)
	avg := 0.0
	for _, v := range primary[skip:] {
		add := math.Pow(math.Abs(v), 1.0/exp)
		if v < 0 {
			avg -= add
		} else {
			avg += add
		}
	}
	avg /= float64(len(primary) - skip)
	thres := math.Pow(math.Abs(avg), exp)
	if avg < 0 {
		thres = -thres
	}
	return thres
}

// thresholdMean returns the mean of the coefficients that set the average
// threshold: primary[skip:] plus any extras, divided by their count. The
// first skip coefficients of primary (typically the DC coefficient) take no
// part in the mean, matching Image::PHash's $skip flag; they still produce
// bits in the caller's bit string.
func thresholdMean(primary []float64, skip int, extra []float64) float64 {
	count := len(primary) + len(extra) - skip
	var sum float64
	for _, v := range primary[skip:] {
		sum += v
	}
	for _, v := range extra {
		sum += v
	}
	return sum / float64(count)
}

// medianThreshold returns the median of the combined primary and extra
// values sorted ascending, with the skip count shrinking the median window
// (the window length is count-skip even though no values are dropped, exactly
// as Perl's `my $len = scalar @vals - $skip`, matching Image::PHash 0.3's
// _apply_median). An odd window takes the middle value; an even window
// averages the two middle values. Extras take part in the threshold but
// produce no bits in the caller's bit string.
func medianThreshold(primary []float64, skip int, extra []float64) float64 {
	vals := make([]float64, 0, len(primary)+len(extra))
	vals = append(vals, primary...)
	vals = append(vals, extra...)
	sort.Float64s(vals)
	n := len(vals) - skip
	if n%2 == 1 {
		return vals[n/2]
	}
	return (vals[n/2-1] + vals[n/2]) / 2
}

// bitsAtOrAbove converts coefficients into a bit string: each primary
// coefficient at index i becomes '1' when it is >= threshold and '0'
// otherwise, mirroring Image::PHash's `$array->[$_] >= $thresh ? '1' : '0'`
// conversion shared by the average, average_x, median, and log methods.
// Every primary coefficient produces a bit, including any excluded from the
// threshold computation by the skip flag.
func bitsAtOrAbove(primary []float64, threshold float64) string {
	var b strings.Builder
	b.Grow(len(primary))
	for _, v := range primary {
		if v >= threshold {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String()
}
