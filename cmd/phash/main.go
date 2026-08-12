// Command phash prints the perceptual hash of an image file as a hex string.
// The default hash uses the library's empty/zero Options defaults (resize 32,
// geometry 8x8, method average), and the --method flag overrides the method,
// mirroring Image::PHash 0.3's option defaults.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/lbe/go-image-phash/internal/consoleslog"
	"github.com/lbe/go-image-phash/phash"
)

func main() {
	os.Exit(run())
}

// run parses the command-line flags, loads the image, and prints its
// perceptual hash, returning the process exit code: 0 on success, 2 when the
// required --path flag is missing (after printing usage), and 1 on load or
// hash errors. Splitting the logic out of main keeps the exit codes explicit
// and the error branches free of os.Exit calls.
func run() int {
	var path, method string
	flag.StringVar(&path, "path", "", "path to the image file to hash")
	flag.StringVar(&method, "method", "", "hash method (average, median, average_x, log, diff); default average")
	flag.Parse()

	logger := consoleslog.NewLogger()

	if path == "" {
		logger.Error("missing required flag", "flag", "path")
		flag.Usage()
		return 2
	}

	// Resize 0 and empty Options select the library defaults, and an empty
	// method keeps the default average method.
	p, err := phash.NewFromPath(path, 0, phash.Options{})
	if err != nil {
		logger.Error("load image", "path", path, "error", err)
		return 1
	}

	h, err := p.PHash(phash.Options{Method: method})
	if err != nil {
		logger.Error("hash image", "path", path, "error", err)
		return 1
	}

	logger.Info("hash computed", "path", path, "hash", h.Hex())
	fmt.Println(h.Hex())
	return 0
}
