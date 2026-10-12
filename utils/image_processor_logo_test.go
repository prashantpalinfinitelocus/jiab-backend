package utils

import (
	"image"
	"image/color"
	"testing"
)

// A solid white logo on a black backdrop shows the logo strength directly: at 25% opacity the pixel is about 64 of 255.
func TestComposeFrame_LogoIsDrawnAtQuarterOpacity(t *testing.T) {
	p := newLayoutProcessor(t)
	cfg := p.config
	backdrop := solidImage{r: image.Rect(0, 0, 10, 10), c: color.RGBA{0, 0, 0, 255}}
	logo := solidImage{r: image.Rect(0, 0, 32, 32), c: color.RGBA{255, 255, 255, 255}}
	meme := solidImage{r: image.Rect(0, 0, cfg.Width, 196), c: layoutTemplate}
	img := p.composeFrame(meme, backdrop, logo, "", "")

	x := cfg.Width - cfg.WatermarkMargin - 3
	y := cfg.Height - cfg.WatermarkMargin - 3
	r, g, b, _ := img.At(x, y).RGBA()
	for _, v := range []uint32{r >> 8, g >> 8, b >> 8} {
		if v < 60 || v > 68 {
			t.Fatalf("logo pixel channel = %d, want about 64 (25%% of 255)", v)
		}
	}
}

func TestWithOpacity_KeepsExistingTransparency(t *testing.T) {
	src := solidImage{r: image.Rect(0, 0, 2, 2), c: color.RGBA{0, 0, 0, 0}}
	out := withOpacity(src, 0.25)
	if _, _, _, a := out.At(0, 0).RGBA(); a != 0 {
		t.Fatalf("transparent pixel became visible: alpha %d", a)
	}
}
