package repointegrity

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const (
	wantModulePath = "github.com/lbe/go-image-phash"

	bannedDepBild        = "github.com/anthonynsimon/bild"
	bannedDepPflag       = "github.com/spf13/pflag"
	bannedDepConsoleSlog = "github.com/phsym/console-slog"
	bannedDepHaxmap      = "github.com/alphadose/haxmap"
)

// bannedDeps are the third-party dependencies that must not appear anywhere in
// the module: go.mod, the resolved module graph, or the import graph of ./...
var bannedDeps = []string{
	bannedDepBild,
	bannedDepPflag,
	bannedDepConsoleSlog,
	bannedDepHaxmap,
}

// TestRepositoryBuildsWithoutBannedDepsOrStubTree verifies that the repository
// builds under the canonical module path with none of the banned dependencies,
// no stub transforms/ or root main.go, an internal/gensyncpool whose package
// tests pass, a dct package free of haxmap whose tests pass, and a phash/
// directory that is either absent or an empty package that compiles.
func TestRepositoryBuildsWithoutBannedDepsOrStubTree(t *testing.T) {
	root, err := moduleRoot()
	if err != nil {
		t.Fatalf("locate module root: %v", err)
	}

	modText, err := readFile(t, filepath.Join(root, "go.mod"))
	if err != nil {
		t.Fatalf("read go.mod: %v", err)
	}

	// Acceptance: go.mod module path is github.com/lbe/go-image-phash.
	if got := modulePath(modText); got != wantModulePath {
		t.Errorf("go.mod module path: got %q, want %q", got, wantModulePath)
	}

	// Acceptance: go.mod references none of the banned dependencies.
	assertNoBannedDeps(t, "go.mod", modText)

	// Acceptance: the resolved module graph (go list -m all) contains none of
	// the banned dependencies.
	if out, err := runGo(t, root, "list", "-m", "all"); err != nil {
		t.Errorf("go list -m all failed: %v\n%s", err, out)
	} else {
		assertNoBannedDeps(t, "go list -m all", out)
	}

	// Acceptance: the import graph of ./... contains none of the banned
	// dependencies, and every package lives under the canonical module path.
	if out, err := runGo(t, root, "list", "-deps", "./..."); err != nil {
		t.Errorf("go list -deps ./... failed: %v\n%s", err, out)
	} else {
		assertNoBannedDeps(t, "go list -deps ./...", out)
		if strings.Contains(out, "go.local/") {
			t.Errorf("package import paths must live under %q, got legacy go.local/ module prefix", wantModulePath)
		}
	}

	// Acceptance: the stub transforms/ directory and root main.go are absent.
	if _, err := os.Stat(filepath.Join(root, "transforms")); err == nil {
		t.Error("stub transforms/ directory must be absent")
	}
	if _, err := os.Stat(filepath.Join(root, "main.go")); err == nil {
		t.Error("stub root main.go must be absent")
	}

	// Acceptance: internal/gensyncpool exists and its package tests pass.
	if fi, err := os.Stat(filepath.Join(root, "internal", "gensyncpool")); err != nil {
		t.Errorf("internal/gensyncpool must exist: %v", err)
	} else if !fi.IsDir() {
		t.Error("internal/gensyncpool must be a directory")
	}
	if out, err := runGo(t, root, "test", "./internal/gensyncpool"); err != nil {
		t.Errorf("go test ./internal/gensyncpool must pass:\n%s", out)
	}

	// Acceptance: dct/static.go no longer uses haxmap (sync.Map or mutex+map
	// instead) and the dct package tests pass.
	if static, err := readFile(t, filepath.Join(root, "dct", "static.go")); err != nil {
		t.Errorf("read dct/static.go: %v", err)
	} else if strings.Contains(static, "haxmap") {
		t.Error("dct/static.go must not reference haxmap; replace it with sync.Map or mutex+map")
	}
	if out, err := runGo(t, root, "test", "./dct"); err != nil {
		t.Errorf("go test ./dct must pass:\n%s", out)
	}

	// Acceptance: phash/ is absent or an empty package that compiles.
	if fi, err := os.Stat(filepath.Join(root, "phash")); err == nil && fi.IsDir() {
		if out, err := runGo(t, root, "test", "./phash"); err != nil {
			t.Errorf("phash/ must be absent or an empty package that compiles; go test ./phash failed:\n%s", out)
		}
	}
}

// assertNoBannedDeps fails the test if any banned dependency appears in text.
func assertNoBannedDeps(t *testing.T, subject, text string) {
	t.Helper()
	for _, dep := range bannedDeps {
		if strings.Contains(text, dep) {
			t.Errorf("%s must not reference banned dependency %q", subject, dep)
		}
	}
}

// moduleRoot returns the nearest ancestor of the working directory that holds
// go.mod (the module root).
func moduleRoot() (string, error) {
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

// modulePath extracts the module path from the go.mod file text.
func modulePath(modText string) string {
	for _, line := range strings.Split(modText, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// readFile reads a file relative to the repository root.
func readFile(t *testing.T, path string) (string, error) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// runGo runs the go tool with the given arguments in the repository root and
// returns its combined output.
func runGo(t *testing.T, root string, args ...string) (string, error) {
	t.Helper()
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	return string(out), err
}
