package phash_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// Reference benchmark results (linux/amd64, Go 1.26.5, Intel Xeon E5-2680 v3):
//
//	BenchmarkPHash_FromPath_Thumbnail-24    	    1044	   1078129 ns/op	   52075 B/op	      30 allocs/op
//	BenchmarkPHash_FromPath_Full-24         	      37	  36219086 ns/op	 1111184 B/op	      34 allocs/op
//	BenchmarkPHash_CachedSecondHash-24      	   70164	     16920 ns/op	    1089 B/op	       2 allocs/op
//	BenchmarkHashFromDCT_8x8-24             	  235736	      4688 ns/op	    1361 B/op	      11 allocs/op
//	BenchmarkHashFromLuminance_32-24        	    8750	    127637 ns/op	   18047 B/op	      14 allocs/op
//	BenchmarkDistance-24                    	 4395400	       241.3 ns/op	       0 B/op	       0 allocs/op
//
// Regenerate with: make bench (see docs/benchmarks.txt).

// distanceBenchA and distanceBenchB are fixed 64-bit hashes (hex
// AAAAAAAAAAAAAAAA vs 0000000000000000).
var (
	distanceBenchA = phash.NewHash(strings.Repeat("1010", 16))
	distanceBenchB = phash.NewHash(strings.Repeat("0000", 16))
)

// distanceSink keeps the benchmark loop observable so the compiler cannot
// fold Distance away.
var distanceSink int

// BenchmarkDistance measures Hamming distance between two fixed 64-bit hashes.
func BenchmarkDistance(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	s := 0
	for i := 0; i < b.N; i++ {
		d, err := phash.Distance(distanceBenchA, distanceBenchB)
		if err != nil {
			b.Fatal(err)
		}
		s += d
	}
	distanceSink = s
}

func benchExamplePath(b *testing.B, name string) string {
	b.Helper()
	path, err := filepath.Abs(filepath.Join("..", "_examples", name))
	if err != nil {
		b.Fatalf("resolve ../_examples/%s: %v", name, err)
	}
	return path
}

// BenchmarkPHash_FromPath_Thumbnail decodes a 32×32 JPEG and hashes it. Resize
// to the default 32×32 is effectively identity work.
func BenchmarkPHash_FromPath_Thumbnail(b *testing.B) {
	path := benchExamplePath(b, "M31_th.jpg")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, err := phash.NewFromPath(path, 0, phash.Options{})
		if err != nil {
			b.Fatal(err)
		}
		if _, err := p.PHash(phash.Options{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPHash_FromPath_Full decodes a large JPEG, scales it to 32×32, and
// hashes it — the typical file-level path where decode and resize dominate.
func BenchmarkPHash_FromPath_Full(b *testing.B) {
	path := benchExamplePath(b, "M31.jpg")
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p, err := phash.NewFromPath(path, 0, phash.Options{})
		if err != nil {
			b.Fatal(err)
		}
		if _, err := p.PHash(phash.Options{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPHash_CachedSecondHash measures a follow-on hash after resize and DCT
// are cached on the PHash instance.
func BenchmarkPHash_CachedSecondHash(b *testing.B) {
	path := benchExamplePath(b, "M31.jpg")
	p, err := phash.NewFromPath(path, 0, phash.Options{})
	if err != nil {
		b.Fatal(err)
	}
	if _, err := p.PHash(phash.Options{}); err != nil {
		b.Fatal(err)
	}

	opts := phash.Options{Geometry: "64", Method: "log"}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := p.PHash(opts); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHashFromDCT_8x8 hashes from a precomputed 32×32 DCT coefficient
// vector with default 8×8 geometry.
func BenchmarkHashFromDCT_8x8(b *testing.B) {
	coeffs := make([]float64, 32*32)
	for i := range coeffs {
		coeffs[i] = float64(i % 97)
	}
	opts := phash.Options{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := phash.HashFromDCT(coeffs, 32, opts); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHashFromLuminance_32 runs luminance → DCT → bit pipeline on a
// synthetic 32×32 block.
func BenchmarkHashFromLuminance_32(b *testing.B) {
	luma := make([]float64, 32*32)
	for i := range luma {
		luma[i] = float64(i % 256)
	}
	opts := phash.Options{}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := phash.HashFromLuminance(luma, 32, opts); err != nil {
			b.Fatal(err)
		}
	}
}
