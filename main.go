package main

import (
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"sync"

	"github.com/anthonynsimon/bild/transform"
	"github.com/phsym/console-slog"
	"github.com/spf13/pflag"
	"golang.org/x/image/bmp"
	"golang.org/x/image/webp"

	"go.local/go-image-phash/dct"
	"go.local/go-image-phash/transforms"
)

var (
	flagPath string
	logger   *slog.Logger
	dctSize  int
)

func setupLogger() {
	logger = slog.New(
		console.NewHandler(
			os.Stderr,
			&console.HandlerOptions{
				Level:      slog.LevelDebug,
				AddSource:  true,
				TimeFormat: "2006-01-02 15:04:05.000000",
			},
		),
	)
	slog.SetDefault(logger)
}

// from ImageHash start
var pixelPool64 = sync.Pool{
	New: func() interface{} {
		p := make([]float64, 4096)
		return &p
	},
}

// pixel2Gray converts a pixel to grayscale value base on luminosity
func pixel2Gray(r, g, b, a uint32) float64 {
	return 0.299*float64(r/257) + 0.587*float64(g/257) + 0.114*float64(b/256)
}

// rgb2GrayDefault uses the image.Image interface
func rgb2GrayDefault(colorImg image.Image, pixels [][]float64, s int) {
	for i := 0; i < s; i++ {
		for j := 0; j < s; j++ {
			logger.Debug("rgb2GrayDefault", "i", i, "j", j)
			f := pixel2Gray(colorImg.At(i, j).RGBA())
			logger.Debug("rgb2GrayDefault", "len(pixels)", len(pixels))
			pixels[i][j] = f
			logger.Debug("breakpoint")
		}
	}
}

// from ImageHash end

//func dctMirror(dct [][]float64) [][]float64 {
//	mirror := make([][]float64,0,64)
//	@mirror = map {($t ^= 1) ? -$_ : $_} @$dct;
//}

func main() {
	// logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	setupLogger()
	logger.Info("Execution starting")

	dctSize = 32

	pflag.StringVarP(&flagPath, "path", "p", "", "source path (file or directory)")
	pflag.Parse()

	// define regexp to match file extension to type
	// const strReImage string = `(?i:bmp|jpe{0,1}g|gif|png|webp)$`
	const strReImageBmp string = `(?i:bmp)$`
	const strReImageJpg string = `(?i:jpe{0,1}g)$`
	const strReImageGif string = `(?i:gif)$`
	const strReImagePng string = `(?i:png)$`
	const strReImageWebp string = `(?i:webp)$`

	// reImage := regexp.MustCompile(strReImage)
	reImageBmp := regexp.MustCompile(strReImageBmp)
	reImageGif := regexp.MustCompile(strReImageGif)
	reImageJpg := regexp.MustCompile(strReImageJpg)
	reImagePng := regexp.MustCompile(strReImagePng)
	reImageWebp := regexp.MustCompile(strReImageWebp)

	// read file
	f, err := os.Open(flagPath)
	if err != nil {
		logger.Error("os.Open", "err", err)
		panic("Open")
	}
	defer f.Close()

	if err != nil {
		logger.Error("os.Open", "err", err)
		panic("Open")
	}
	extType := ""

	switch {
	case reImageBmp.MatchString(flagPath):
		extType = "image/bmp"
	case reImageGif.MatchString(flagPath):
		extType = "image/gif"
	case reImageJpg.MatchString(flagPath):
		extType = "image/jpeg"
	case reImagePng.MatchString(flagPath):
		extType = "image/png"
	case reImageWebp.MatchString(flagPath):
		extType = "image/webp"
	default:
		{
			logger.Error("Unmatched path", "path", flagPath)
			return
		}
	}

	l := 512
	buf := make([]byte, l)
	if _, err := io.ReadAtLeast(f, buf, l); err != nil {
		if err == io.EOF {
			logger.Error("Unexpected EOF", "err", err, "path", flagPath)
			panic("io.ReadAtLeast")
		}
		//if err == io.ErrUnexpectedEOF {
		//	//logger.Error("Premature EOF", "err", err, "n", n, "path", flagPath)
		//	return
		//}
	}
	contentType := http.DetectContentType(buf)
	if _, err := f.Seek(0, 0); err != nil {
		logger.Error("processFile seek", "err", err)
		panic("processFile")
	}

	if extType != contentType {
		logger.Warn("Conflicting extension and content type", "contentType", contentType, "path", flagPath)
	}

	var img image.Image
	switch contentType {
	case "image/bmp": // reImageBmp.MatchString(flagPath):
		{
			if img, err = bmp.Decode(f); err != nil {
				logger.Error("processFile bmp.Decode", "err", err, "path", flagPath)
				return
			}
		}
	case "image/gif": // reImageGif.MatchString(flagPath):
		{
			if img, err = gif.Decode(f); err != nil {
				logger.Error("processFile gif.Decode", "err", err, "path", flagPath)
				return
			}
		}
	case "image/jpeg": // reImageJpg.MatchString(flagPath):
		{
			if img, err = jpeg.Decode(f); err != nil {
				logger.Error("processFile jpeg.Decode", "err", err, "path", flagPath)
				return
			}
		}
	case "image/png": // reImagePng.MatchString(flagPath):
		{
			if img, err = png.Decode(f); err != nil {
				logger.Error("processFile png.Decode", "err", err, "path", flagPath)
				return
			}
		}
	case "image/webp": // reImageWebp.MatchString(flagPath):
		{
			if img, err = webp.Decode(f); err != nil {
				logger.Error("processFile webp.Decode", "err", err, "path", flagPath)
				return
			}
		}
	default:
		{
			logger.Error("Unsupported Content Type", "contentType", contentType, "path", flagPath)
			return
		}
	}

	// scale image if not dctSize x dctSize
	imgScaled := img
	bounds := imgScaled.Bounds()
	w, h := bounds.Max.X-bounds.Min.X, bounds.Max.Y-bounds.Min.Y
	if w != dctSize || h != dctSize {
		imgScaled = transform.Resize(img, dctSize, dctSize, transform.NearestNeighbor)
	}

	// convert from RGB to gray scale
	pixels := pixelPool64.Get().(*[]float64)
	transforms.Rgb2GrayFast(imgScaled, pixels)
	flattens := dct.DCT_2D(*pixels, dctSize)

	logger.Debug("ou:t", "flattens", flattens)

	logger.Info("Execution Complete")
}
