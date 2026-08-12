//go:build perl

package phash_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/lbe/go-image-phash/dct"
	"github.com/lbe/go-image-phash/phash"
)

// TestPerlCrossChecksAlgorithmicGoldens is the acceptance for the optional perl
// build-tag suite: when Image::PHash is installed in Perl it cross-checks only
// the algorithmic goldens — the DCT coefficients (coeffs), the
// luminance-block hashes (luminance), the DCT-fixture bit pipeline (pipeline),
// the bits-to-hex conversion (b2h), and the Hamming distance (diff) — by
// re-deriving each expected value from the live Perl module and comparing it
// with the Go implementation, so any drift between the two fails the test.
// Without Perl/Image::PHash the whole test skips cleanly in requirePerlPHash, so
// `go test -tags=perl ./...` stays green on machines that do not have the Perl
// module installed.
//
// The Go file-level goldens from cycle 30 (testdata/files/files.json and the
// _examples hashes) are deliberately NOT cross-checked here: the Go pipeline
// resizes with ApproxBiLinear while Image::PHash 0.3 uses Imager's mixing
// resize, so for images larger than 32x32 the file-level hashes legitimately
// differ from Perl (only the already-32x32 M31_th.jpg matches), as
// testdata/files/files.json's about text documents. Only the algorithmic steps
// that are bit-compatible with Perl are cross-checked.
func TestPerlCrossChecksAlgorithmicGoldens(t *testing.T) {
	perl := requirePerlPHash(t)

	t.Run("coeffs", func(t *testing.T) {
		// The raw DCT step: Perl computes Math::DCT dct2d over the same
		// luminance block the luminance goldens hash, and the Go side runs
		// the exported DCT_2D over it. Coefficient equality is
		// tolerance-based because both sides are floating point, but the
		// tolerance still rejects any drift that could flip a threshold
		// decision downstream (those decisions are pinned bit-exactly by the
		// luminance and pipeline subtests).
		luminance, resize, _ := loadLuminanceGoldens(t)

		fixture := mustJSON(t, map[string]any{
			"luminance": luminance,
			"resize":    resize,
		})
		var out struct {
			Coeffs []float64 `json:"coeffs"`
		}
		mustDecodeJSON(t, perl.runScript(t, coeffsScript, fixture), &out)

		if len(out.Coeffs) != len(luminance) {
			t.Fatalf("Perl returned %d DCT coefficients, want %d (resize %d)", len(out.Coeffs), len(luminance), resize)
		}

		input := append([]float64(nil), luminance...)
		coeffs := make([]float64, len(input))
		dct.DCT_2D(&input, resize, &coeffs)

		const coeffTolerance = 1e-6
		maxDiff := 0.0
		var mismatches []string
		for i := range coeffs {
			diff := math.Abs(coeffs[i] - out.Coeffs[i])
			if diff > maxDiff {
				maxDiff = diff
			}
			if diff > coeffTolerance {
				mismatches = append(mismatches, fmt.Sprintf("coefficient %d: Go %v vs Perl %v (diff %g)", i, coeffs[i], out.Coeffs[i], diff))
			}
		}
		if len(mismatches) > 0 {
			t.Errorf("Go DCT_2D drifted from the live Image::PHash DCT over the luminance fixture: %d of %d coefficients differ by more than %g (max diff %g); first mismatches: %s",
				len(mismatches), len(coeffs), coeffTolerance, maxDiff, strings.Join(mismatches[:min(len(mismatches), 5)], "; "))
		}
	})

	t.Run("luminance", func(t *testing.T) {
		// The luminance golden end-to-end: the fixture luminance block, run
		// through Perl's own dct2d and scalar-context pHash pipeline, must
		// produce the same hex as the Go HashFromLuminance composition. The
		// Perl side also feeds the checked-in golden its own hex, so a changed
		// Perl module version surfaces as drift too.
		luminance, resize, cases := loadLuminanceGoldens(t)

		fixture := mustJSON(t, map[string]any{
			"luminance": luminance,
			"resize":    resize,
			"cases":     cases,
		})
		var out struct {
			Results []perlHashCase `json:"results"`
		}
		mustDecodeJSON(t, perl.runScript(t, luminanceScript, fixture), &out)

		if len(out.Results) != len(cases) {
			t.Fatalf("Perl returned %d luminance hashes, want %d", len(out.Results), len(cases))
		}
		for i, c := range cases {
			opts := phash.Options{Geometry: c.Geometry, Reduce: c.Reduce, Method: c.Method, Mirror: c.Mirror, MirrorProof: c.MirrorProof}
			got, err := phash.HashFromLuminance(luminance, resize, opts)
			if err != nil {
				t.Errorf("case %q: HashFromLuminance(%+v) returned an unexpected error: %v", c.Name, opts, err)
				continue
			}
			want := out.Results[i].Hex
			if got.Hex() != want {
				t.Errorf("case %q: Go HashFromLuminance hex %q drifted from live Perl %q", c.Name, got.Hex(), want)
			}
			if want != c.Hex {
				t.Errorf("case %q: live Perl hex %q diverged from the checked-in luminance golden %q — the Perl module version or the golden drifted", c.Name, want, c.Hex)
			}
		}
	})

	t.Run("pipeline", func(t *testing.T) {
		// The DCT-fixture pipeline golden end-to-end: the Perl pHash over the
		// fixed DCT matrix must produce the same hex as the Go HashFromDCT bit
		// pipeline for every golden case.
		dctFixture, resize, cases := loadPipelineGoldens(t)

		fixture := mustJSON(t, map[string]any{
			"dct":    dctFixture,
			"resize": resize,
			"cases":  cases,
		})
		var out struct {
			Results []perlHashCase `json:"results"`
		}
		mustDecodeJSON(t, perl.runScript(t, pipelineScript, fixture), &out)

		if len(out.Results) != len(cases) {
			t.Fatalf("Perl returned %d pipeline hashes, want %d", len(out.Results), len(cases))
		}
		for i, c := range cases {
			opts := phash.Options{Geometry: c.Geometry, Reduce: c.Reduce, Method: c.Method, Mirror: c.Mirror, MirrorProof: c.MirrorProof}
			got, err := phash.HashFromDCT(dctFixture, resize, opts)
			if err != nil {
				t.Errorf("case %q: HashFromDCT(%+v) returned an unexpected error: %v", c.Name, opts, err)
				continue
			}
			want := out.Results[i].Hex
			if got.Hex() != want {
				t.Errorf("case %q: Go HashFromDCT hex %q drifted from live Perl %q", c.Name, got.Hex(), want)
			}
			if want != c.Hex {
				t.Errorf("case %q: live Perl hex %q diverged from the checked-in pipeline golden %q — the Perl module version or the golden drifted", c.Name, want, c.Hex)
			}
		}
	})

	t.Run("b2h", func(t *testing.T) {
		// The b2h golden inputs, run through Perl's own b2h and the Go
		// BitsToHex: hex output must agree for every bit string, including the
		// lengths not divisible by 4.
		goldens := loadB2hGoldens(t)

		bits := make([]string, len(goldens))
		for i, c := range goldens {
			bits[i] = c.bits
		}
		fixture := mustJSON(t, map[string]any{"bits": bits})
		var out struct {
			Results []struct {
				Bits string `json:"bits"`
				Hex  string `json:"hex"`
			} `json:"results"`
		}
		mustDecodeJSON(t, perl.runScript(t, b2hScript, fixture), &out)

		if len(out.Results) != len(goldens) {
			t.Fatalf("Perl returned %d b2h results, want %d", len(out.Results), len(goldens))
		}
		for i, c := range goldens {
			got := phash.BitsToHex(c.bits)
			want := out.Results[i].Hex
			if got != want {
				t.Errorf("BitsToHex(%q) = %q, drifted from live Perl b2h %q", c.bits, got, want)
			}
			if want != c.want {
				t.Errorf("live Perl b2h(%q) = %q diverged from the checked-in golden %q — the Perl module version or the golden drifted", c.bits, want, c.want)
			}
		}
	})

	t.Run("diff", func(t *testing.T) {
		// The Hamming distance: Image::PHash::diff over hex hash pairs must
		// agree with Go Distance over the equivalent bit strings. The first
		// pairs are the ones Perl t/util.t and phash/distance_test.go pin
		// (4-bit, >64-bit, identical, all-bits-differ), so their known
		// distances also catch a Perl module version that drifted; the last
		// pair is two real 64-bit hashes from the pipeline goldens with no
		// recorded distance, where Go and Perl must still agree.
		_, _, pipelineCases := loadPipelineGoldens(t)
		var defaultAvgHex, medianHex string
		for _, c := range pipelineCases {
			switch c.Name {
			case "default 8x8 average":
				defaultAvgHex = c.Hex
			case "8x8 median":
				medianHex = c.Hex
			}
		}
		if defaultAvgHex == "" || medianHex == "" {
			t.Fatal("pipeline goldens are missing the default 8x8 average or 8x8 median cases")
		}

		pairs := []struct {
			name string
			a, b string
			want int // known distance, or -1 when only Go-vs-Perl agreement is checked
		}{
			{"perl t/util.t diff of hex A and B (4 bits)", "A", "B", 1},
			{"perl t/util.t diff of the 17-hex-digit pair (68 bits)", "AAAAAAAAAAAAAAAAA", "AAAAAAAAAAAAAAAAB", 1},
			{"identical 64-bit hashes", "AAAAAAAAAAAAAAAA", "AAAAAAAAAAAAAAAA", 0},
			{"all 64 bits differ", "FFFFFFFFFFFFFFFF", "0000000000000000", 64},
			{"pipeline golden default 8x8 average vs 8x8 median", defaultAvgHex, medianHex, -1},
		}

		type perlPair struct {
			A string `json:"a"`
			B string `json:"b"`
		}
		inputs := make([]perlPair, len(pairs))
		for i, p := range pairs {
			inputs[i] = perlPair{A: p.a, B: p.b}
		}
		fixture := mustJSON(t, map[string]any{"pairs": inputs})
		var out struct {
			Results []struct {
				A        string `json:"a"`
				B        string `json:"b"`
				Distance int    `json:"distance"`
			} `json:"results"`
		}
		mustDecodeJSON(t, perl.runScript(t, diffScript, fixture), &out)

		if len(out.Results) != len(pairs) {
			t.Fatalf("Perl returned %d diff results, want %d", len(out.Results), len(pairs))
		}
		for i, p := range pairs {
			got, err := phash.Distance(phash.NewHash(hexToBits(t, p.a)), phash.NewHash(hexToBits(t, p.b)))
			if err != nil {
				t.Errorf("case %q: Distance(%q, %q) returned an unexpected error: %v", p.name, p.a, p.b, err)
				continue
			}
			want := out.Results[i].Distance
			if got != want {
				t.Errorf("case %q: Go Distance %d drifted from live Perl diff %d (hex %q vs %q)", p.name, got, want, p.a, p.b)
			}
			if p.want >= 0 && want != p.want {
				t.Errorf("case %q: live Perl diff %d diverged from the known distance %d — the Perl module version drifted", p.name, want, p.want)
			}
		}
	})
}

// perlHashCase is one result row the Perl scripts emit for the luminance/pipeline
// case lists: the case name and the scalar-context pHash hex.
type perlHashCase struct {
	Name string `json:"name"`
	Hex  string `json:"hex"`
}

// perlPHash is a working perl interpreter that can load Image::PHash, found by
// requirePerlPHash. inc holds the -I include arguments every Perl invocation
// must pass so the module resolves the same way the probe found it.
type perlPHash struct {
	perl string
	inc  []string
}

// requirePerlPHash locates a perl interpreter that can load Image::PHash and
// returns it (plus the -I include arguments that made the probe succeed), or
// skips the test cleanly when no usable Perl installation exists. The probe
// tries, in order: the perl's own @INC (the module installed in the active
// environment, e.g. via PERL5LIB), an explicit IMAGE_PHASH_LIB override, and the
// sibling Image-PHash checkout the plan documents ("checkout at
// ../src/Image-PHash" — from this package dir that is ../../Image-PHash). A
// missing perl, a perl without Image::PHash in @INC, or a perl that cannot
// load the module (e.g. a missing Math::DCT dependency) all skip, so
// `go test -tags=perl ./...` passes cleanly on machines without Perl.
func requirePerlPHash(t *testing.T) perlPHash {
	t.Helper()

	perlPath := os.Getenv("PERL")
	if perlPath == "" {
		var err error
		perlPath, err = exec.LookPath("perl")
		if err != nil {
			t.Skipf("perl is not on PATH (and PERL is unset): skipping the Image::PHash Perl cross-checks")
		}
	}

	type candidate struct {
		inc  []string
		desc string
	}
	candidates := []candidate{{desc: fmt.Sprintf("perl %s with its own @INC", perlPath)}}
	if lib := os.Getenv("IMAGE_PHASH_LIB"); lib != "" {
		candidates = append(candidates, candidate{inc: []string{"-I", lib}, desc: fmt.Sprintf("IMAGE_PHASH_LIB=%s", lib)})
	}
	if lib, err := filepath.Abs(filepath.Join("..", "..", "Image-PHash", "lib")); err == nil {
		if _, err := os.Stat(filepath.Join(lib, "Image", "PHash.pm")); err == nil {
			candidates = append(candidates, candidate{inc: []string{"-I", lib}, desc: "the sibling Image-PHash checkout " + lib})
		}
	}

	var lastProbe []byte
	for _, c := range candidates {
		args := append(append([]string{}, c.inc...), "-MImage::PHash", "-e", "print $Image::PHash::VERSION")
		out, err := exec.Command(perlPath, args...).CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return perlPHash{perl: perlPath, inc: c.inc}
		}
		lastProbe = out
	}

	descs := make([]string, len(candidates))
	for i, c := range candidates {
		descs[i] = c.desc
	}
	t.Skipf("no usable Image::PHash installation: probed %s with no success (last probe output: %s); skipping the perl build-tag cross-checks — install Image::PHash + Math::DCT for the active perl, or point PERL/IMAGE_PHASH_LIB at them",
		strings.Join(descs, ", "), strings.TrimSpace(string(lastProbe)))
	return perlPHash{}
}

// runScript writes script to a temp file, pipes the fixture JSON to it on stdin,
// and returns the Perl script's stdout. The script must print one JSON document.
func (o perlPHash) runScript(t *testing.T, script string, fixture []byte) []byte {
	t.Helper()

	scriptPath := filepath.Join(t.TempDir(), "perl-crosscheck.pl")
	if err := os.WriteFile(scriptPath, []byte(script), 0o600); err != nil {
		t.Fatalf("write Perl cross-check script: %v", err)
	}

	args := append(append([]string{}, o.inc...), scriptPath)
	cmd := exec.Command(o.perl, args...)
	cmd.Stdin = bytes.NewReader(fixture)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("Perl cross-check script failed: %v\nstderr:\n%s", err, stderr.String())
	}
	return stdout.Bytes()
}

// mustJSON marshals v to JSON, failing the test on error.
func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal Perl fixture: %v", err)
	}
	return data
}

// mustDecodeJSON decodes the Perl script's JSON stdout into v, failing the test
// on malformed output.
func mustDecodeJSON(t *testing.T, data []byte, v any) {
	t.Helper()
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("decode Perl output %q: %v", data, err)
	}
}

// hexToBits expands a hex string into its 4-bits-per-digit bit string, the
// representation phash.Hash carries. Only nibble-aligned hashes are fed to the
// diff cross-check, so the expansion is exact.
func hexToBits(t *testing.T, hex string) string {
	t.Helper()
	var b strings.Builder
	for _, c := range hex {
		v, err := strconv.ParseUint(string(c), 16, 4)
		if err != nil {
			t.Fatalf("invalid hex digit %q in %q: %v", c, hex, err)
		}
		fmt.Fprintf(&b, "%04b", v)
	}
	return b.String()
}

// b2hGoldenCase is one input/output pair from testdata/b2h/golden.txt.
type b2hGoldenCase struct {
	bits string
	want string
}

// loadB2hGoldens parses testdata/b2h/golden.txt (the checked-in b2h golden
// vectors, generated from the verbatim Image::PHash 0.3 b2h subroutine) into
// its <bits>\t<hex> lines, skipping comments and blanks.
func loadB2hGoldens(t *testing.T) []b2hGoldenCase {
	t.Helper()
	f, err := os.Open("../testdata/b2h/golden.txt")
	if err != nil {
		t.Fatalf("open b2h golden vectors: %v", err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Errorf("close b2h golden vectors: %v", err)
		}
	})

	var cases []b2hGoldenCase
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) != 2 {
			t.Fatalf("malformed b2h golden line %q", line)
		}
		cases = append(cases, b2hGoldenCase{bits: fields[0], want: fields[1]})
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan b2h golden vectors: %v", err)
	}
	if len(cases) == 0 {
		t.Fatal("no b2h golden cases loaded")
	}
	return cases
}

// coeffsScript computes the 2D DCT of the fixture luminance block with
// Image::PHash's Math::DCT dct2d, the exact call Image::PHash 0.3's pHash
// method makes when no DCT is cached.
const coeffsScript = `use strict;
use warnings;
use Math::DCT qw/dct2d/;
use JSON::PP;

my $data = JSON::PP->new->decode(do { local $/; <STDIN> });
my $coeffs = dct2d($data->{luminance}, $data->{resize});
print JSON::PP->new->encode({ coeffs => $coeffs });
`

// luminanceScript drives the real Image::PHash 0.3 pHash method through a fake
// $self carrying the fixture luminance block as pixels and no cached dct, so
// Perl computes Math::DCT dct2d itself before the bit pipeline — the same flow
// the checked-in luminance goldens were generated with
// (tmp/golden-gen/luminance-golden.pl).
const luminanceScript = `use strict;
use warnings;
use Image::PHash;
use JSON::PP;

my $data = JSON::PP->new->decode(do { local $/; <STDIN> });
my $self = bless {
    dct_size  => $data->{resize},
    im_scaled => 1,
    methods   => {
        average   => \&Image::PHash::_apply_average,
        average_x => \&Image::PHash::_apply_average,
        median    => \&Image::PHash::_apply_median,
        diff      => \&Image::PHash::_apply_diff,
        log       => \&Image::PHash::_apply_log_average,
    },
    pixels  => sub { return $data->{luminance} },
    reduced => sub { },
}, 'Image::PHash';

my @out;
for my $c (@{ $data->{cases} }) {
    my %opt;
    $opt{geometry}    = $c->{geometry};
    $opt{method}      = $c->{method};
    $opt{reduce}      = 1 if $c->{reduce};
    $opt{mirror}      = 1 if $c->{mirror};
    $opt{mirrorproof} = 1 if $c->{mirrorproof};
    my $hex = $self->pHash(%opt);
    push @out, { name => $c->{name}, hex => $hex };
}
print JSON::PP->new->encode({ results => \@out });
`

// pipelineScript drives the real Image::PHash 0.3 pHash method through a fake
// $self carrying the fixture DCT matrix as its cached dct (im_scaled set, so the
// reduce step is skipped), the same flow the checked-in pipeline goldens were
// generated with (tmp/gen_pipeline_goldens.pl). The pixels stub still has to
// be callable because pHash evaluates the dct2d argument even when the DCT is
// cached.
const pipelineScript = `use strict;
use warnings;
use Image::PHash;
use JSON::PP;

my $data = JSON::PP->new->decode(do { local $/; <STDIN> });
my $self = bless {
    dct_size  => $data->{resize},
    im_scaled => 1,
    dct       => $data->{dct},
    methods   => {
        average   => \&Image::PHash::_apply_average,
        average_x => \&Image::PHash::_apply_average,
        median    => \&Image::PHash::_apply_median,
        diff      => \&Image::PHash::_apply_diff,
        log       => \&Image::PHash::_apply_log_average,
    },
    pixels  => sub { return [] },
    reduced => sub { },
}, 'Image::PHash';

my @out;
for my $c (@{ $data->{cases} }) {
    my %opt;
    $opt{geometry}    = $c->{geometry};
    $opt{method}      = $c->{method};
    $opt{reduce}      = 1 if $c->{reduce};
    $opt{mirror}      = 1 if $c->{mirror};
    $opt{mirrorproof} = 1 if $c->{mirrorproof};
    my $hex = $self->pHash(%opt);
    push @out, { name => $c->{name}, hex => $hex };
}
print JSON::PP->new->encode({ results => \@out });
`

// b2hScript runs the verbatim Image::PHash 0.3 b2h subroutine over each input
// bit string.
const b2hScript = `use strict;
use warnings;
use Image::PHash;
use JSON::PP;

my $data = JSON::PP->new->decode(do { local $/; <STDIN> });
my @out;
for my $bits (@{ $data->{bits} }) {
    push @out, { bits => $bits, hex => Image::PHash::b2h($bits) };
}
print JSON::PP->new->encode({ results => \@out });
`

// diffScript runs the verbatim Image::PHash 0.3 diff subroutine over each hex
// hash pair.
const diffScript = `use strict;
use warnings;
use Image::PHash;
use JSON::PP;

my $data = JSON::PP->new->decode(do { local $/; <STDIN> });
my @out;
for my $p (@{ $data->{pairs} }) {
    push @out, { a => $p->{a}, b => $p->{b}, distance => Image::PHash::diff($p->{a}, $p->{b}) };
}
print JSON::PP->new->encode({ results => \@out });
`
