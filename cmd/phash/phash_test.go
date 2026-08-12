package main_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/lbe/go-image-phash/phash"
)

// exampleJPG is the checked-in JPEG the acceptance command hashes.
const exampleJPG = "_examples/M31.jpg"

// bannedDeps are the third-party dependencies the CLI must not introduce: the
// plan commits the CLI to the stdlib flag package (no pflag) and to an
// internal console slog handler (no phsym/console-slog).
var bannedDeps = []string{
	"github.com/spf13/pflag",
	"github.com/phsym/console-slog",
}

// hexTokenRe matches a run of hex digits, the shape of a phash hex hash.
var hexTokenRe = regexp.MustCompile(`[0-9A-Fa-f]+`)

// TestPHashCLIPrintsHashForImagePath is the e2e acceptance for cmd/phash: the
// acceptance command `go run ./cmd/phash --path _examples/M31.jpg` exits 0 and
// prints a non-empty hex hash computed with the Image::PHash 0.3 empty/zero
// Options defaults (resize 32, geometry 8x8, method average) unless flags
// override them; the JPEG decode format is registered (M31.jpg decodes); the
// CLI logs through the internal consoleslog package; and the module carries no
// pflag or phsym/console-slog dependency.
func TestPHashCLIPrintsHashForImagePath(t *testing.T) {
	root := moduleRoot(t)

	// Acceptance: the module has no pflag or phsym/console-slog dependency —
	// neither in go.mod nor in the resolved module graph.
	modText := readFile(t, root, "go.mod")
	for _, dep := range bannedDeps {
		if strings.Contains(modText, dep) {
			t.Errorf("go.mod must not reference banned dependency %q", dep)
		}
	}
	if out, err := runGo(t, root, "list", "-m", "all"); err != nil {
		t.Errorf("go list -m all must succeed: %v\n%s", err, out)
	} else {
		for _, dep := range bannedDeps {
			if strings.Contains(out, dep) {
				t.Errorf("resolved module graph must not contain banned dependency %q", dep)
			}
		}
	}

	// Acceptance: the CLI logs through the internal consoleslog package, so
	// the console handler lives in the module instead of phsym/console-slog.
	if _, err := os.Stat(filepath.Join(root, "internal", "consoleslog")); err != nil {
		t.Errorf("internal/consoleslog package must exist (cmd/phash logs through the internal console slog handler): %v", err)
	} else if out, err := runGo(t, root, "build", "./internal/consoleslog"); err != nil {
		t.Errorf("internal/consoleslog must build: %v\n%s", err, out)
	}

	t.Run("prints the zero-Options default hash for an image path", func(t *testing.T) {
		stdout, _ := runPHashCLI(t, root, "--path", exampleJPG)
		got := extractHexHash(t, stdout)
		want := hashViaLibrary(t, root, phash.Options{})
		if got != want {
			t.Errorf("go run ./cmd/phash --path %s: got hex hash %q, want %q (Image::PHash 0.3 empty/zero Options defaults: resize 32, geometry 8x8, method average)", exampleJPG, got, want)
		}
	})

	t.Run("flags override the Image::PHash 0.3 defaults", func(t *testing.T) {
		stdout, _ := runPHashCLI(t, root, "--path", exampleJPG, "--method", "median")
		got := extractHexHash(t, stdout)
		want := hashViaLibrary(t, root, phash.Options{Method: "median"})
		if got != want {
			t.Errorf("go run ./cmd/phash --path %s --method median: got hex hash %q, want %q (the flag must override the default average method)", exampleJPG, got, want)
		}
	})
}

// moduleRoot returns the repository root, the nearest ancestor of the working
// directory that holds go.mod.
func moduleRoot(t *testing.T) string {
	t.Helper()
	root, err := findModuleRoot()
	if err != nil {
		t.Fatalf("locate module root: %v", err)
	}
	return root
}

// findModuleRoot walks up from the working directory to the nearest directory
// holding go.mod.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found at or above %s", dir)
		}
		dir = parent
	}
}

// readFile returns the contents of the file at path relative to root.
func readFile(t *testing.T, root, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

// runGo runs the go tool with args in root and returns its combined output.
func runGo(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// runPHashCLI runs the acceptance command `go run ./cmd/phash [args...]` in
// root and returns its stdout and stderr separately. A non-zero exit is
// reported as the CLI failing the acceptance, not as a helper setup problem.
func runPHashCLI(t *testing.T, root string, args ...string) (string, string) {
	t.Helper()
	cmd := exec.Command("go", append([]string{"run", "./cmd/phash"}, args...)...)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("go run ./cmd/phash %s must exit 0 and print a non-empty hex hash; got error: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String(), stderr.String()
}

// extractHexHash returns the longest run of hex digits in stdout, uppercased:
// a phash hex hash is a run of hex digits, and the longest such run is the
// hash whether the CLI prints it alone or alongside a label.
func extractHexHash(t *testing.T, stdout string) string {
	t.Helper()
	best := ""
	for _, m := range hexTokenRe.FindAllString(stdout, -1) {
		if len(m) > len(best) {
			best = m
		}
	}
	if best == "" {
		t.Fatalf("stdout must contain a non-empty hex hash; got %q", stdout)
	}
	return strings.ToUpper(best)
}

// hashViaLibrary returns the hex hash the phash library computes for the
// example image with the given options — the CLI output must match it exactly.
func hashViaLibrary(t *testing.T, root string, opts phash.Options) string {
	t.Helper()
	p, err := phash.NewFromPath(filepath.Join(root, exampleJPG), 0, opts)
	if err != nil {
		t.Fatalf("phash.NewFromPath(%q, 0, %+v) must succeed: %v", exampleJPG, opts, err)
	}
	h, err := p.PHash(opts)
	if err != nil {
		t.Fatalf("PHash(%+v) must succeed: %v", opts, err)
	}
	return h.Hex()
}
