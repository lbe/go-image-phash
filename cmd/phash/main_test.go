package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"
)

func TestRunMissingPathReturnsUsageExitCode(t *testing.T) {
	resetFlags(t)
	if code := runWithArgs(t, "phash"); code != 2 {
		t.Fatalf("run() without -path = %d, want 2", code)
	}
}

func TestRunHashesExampleImage(t *testing.T) {
	root := moduleRoot(t)
	resetFlags(t)
	code := runWithArgs(t, "phash", "-path", filepath.Join(root, "_examples/M31.jpg"))
	if code != 0 {
		t.Fatalf("run() with valid path = %d, want 0", code)
	}
}

func TestRunUnknownPathReturnsErrorExitCode(t *testing.T) {
	resetFlags(t)
	if code := runWithArgs(t, "phash", "-path", "/no/such/image.jpg"); code != 1 {
		t.Fatalf("run() with missing file = %d, want 1", code)
	}
}

func TestRunInvalidMethodReturnsErrorExitCode(t *testing.T) {
	root := moduleRoot(t)
	resetFlags(t)
	if code := runWithArgs(t, "phash", "-path", filepath.Join(root, "_examples/M31.jpg"), "-method", "bogus"); code != 1 {
		t.Fatalf("run() with invalid -method = %d, want 1", code)
	}
}

func TestRunMethodFlagOverridesDefault(t *testing.T) {
	root := moduleRoot(t)
	resetFlags(t)
	if code := runWithArgs(t, "phash", "-path", filepath.Join(root, "_examples/M31.jpg"), "-method", "median"); code != 0 {
		t.Fatalf("run() with -method median = %d, want 0", code)
	}
}

func resetFlags(t *testing.T) {
	t.Helper()
	flag.CommandLine = flag.NewFlagSet("phash", flag.ContinueOnError)
}

func runWithArgs(t *testing.T, args ...string) int {
	t.Helper()
	os.Args = args
	return run()
}

func moduleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}
