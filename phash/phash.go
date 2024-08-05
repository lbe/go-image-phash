package phash

import (
	"image"
	"log/slog"
)

var logger *slog.Logger

type tPHash struct {
	Dct []float64
	DctMirror []float64
	DctSize int
	Geometry string
	ImgBase image.Image
	ImgReduced image.Image
	imgPath string
	ImgType string
	Mirror bool
	MirrorProof bool
	phash []int
	Pixels []int
	Reduce bool
}

func (p *tPHash) Phash() []int {

}

func (p *tPHash) Load(imgPath string)