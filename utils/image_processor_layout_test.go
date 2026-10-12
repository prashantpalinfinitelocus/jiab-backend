package utils

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

// The checks use a magenta template on a dark green backdrop, so any white (text fill) or black
// (text outline) pixel can only have come from a caption.
var (
	layoutBackdrop = color.RGBA{0, 80, 40, 255}
	layoutTemplate = color.RGBA{200, 0, 200, 255}
)

func newLayoutProcessor(t *testing.T) *ImageProcessor {
	t.Helper()
	p, err := NewImageProcessor(DefaultImageProcessorConfig("", ""), "")
	if err != nil {
		t.Fatalf("processor: %v", err)
	}
	return p
}

func compose(t *testing.T, p *ImageProcessor, w, h int, top, bottom string) image.Image {
	t.Helper()
	backdrop := solidImage{r: image.Rect(0, 0, 10, 10), c: layoutBackdrop}
	logo := solidImage{r: image.Rect(0, 0, 32, 32), c: layoutBackdrop} // same colour as the backdrop: adds no text-coloured pixels
	meme := solidImage{r: image.Rect(0, 0, w, h), c: layoutTemplate}
	return p.composeFrame(meme, backdrop, logo, top, bottom)
}

func isTextPixel(c color.Color) bool {
	r, g, b, _ := c.RGBA()
	r, g, b = r>>8, g>>8, b>>8
	return (r > 240 && g > 240 && b > 240) || (r < 10 && g < 10 && b < 10)
}

// textPixels counts caption-coloured pixels in the rows [y0, y1).
func textPixels(img image.Image, y0, y1 int) int {
	n := 0
	for y := y0; y < y1; y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			if isTextPixel(img.At(x, y)) {
				n++
			}
		}
	}
	return n
}

func TestComposeFrame_TextStaysInItsBandsAndTemplateStaysClean(t *testing.T) {
	p := newLayoutProcessor(t)
	cfg := p.config
	midTop := cfg.TopBandHeight
	midBottom := cfg.Height - cfg.BottomBandHeight
	textBoxBottom := midBottom + cfg.BottomTextBox

	long := strings.Repeat("A VERY LONG CAPTION THAT WOULD NEED MANY LINES ", 3) // far more than two lines
	cases := map[string]struct{ top, bottom string }{
		"short":    {"TOP", "BOTTOM"},
		"two line": {"THIS TOP CAPTION IS LONG ENOUGH TO WRAP ONTO TWO LINES", "THIS BOTTOM CAPTION IS ALSO LONG ENOUGH TO WRAP"},
		"overflow": {long, long},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			out := compose(t, p, 480, 270, c.top, c.bottom)
			if out.Bounds().Dx() != cfg.Width || out.Bounds().Dy() != cfg.Height {
				t.Fatalf("size = %v, want %dx%d", out.Bounds(), cfg.Width, cfg.Height)
			}
			if got := textPixels(out, midTop, midBottom); got != 0 {
				t.Errorf("%d caption pixels landed on the template area (rows %d-%d)", got, midTop, midBottom)
			}
			if textPixels(out, 0, midTop) == 0 {
				t.Error("top caption is missing from the top band")
			}
			if textPixels(out, midBottom, textBoxBottom) == 0 {
				t.Error("bottom caption is missing from the bottom band")
			}
			if got := textPixels(out, textBoxBottom, cfg.Height); got != 0 {
				t.Errorf("%d caption pixels spilled into the logo area below the caption box", got)
			}
		})
	}
}

func TestComposeFrame_TemplateIsCentredInTheMiddle(t *testing.T) {
	p := newLayoutProcessor(t)
	cfg := p.config
	middle := cfg.Height - cfg.TopBandHeight - cfg.BottomBandHeight

	// A wide template fills the middle edge to edge.
	wide := compose(t, p, 480, 270, "", "")
	if got := wide.At(cfg.Width/2, cfg.TopBandHeight+middle/2); !sameColour(got, layoutTemplate) {
		t.Errorf("middle of a wide template = %v, want the template colour", got)
	}
	if got := wide.At(cfg.Width/2, cfg.TopBandHeight/2); !sameColour(got, layoutBackdrop) {
		t.Errorf("top band without a caption = %v, want the plain backdrop", got)
	}

	// A tall template is fitted to the middle height and centred, leaving backdrop at the sides.
	tall := compose(t, p, 270, 480, "", "")
	if got := tall.At(cfg.Width/2, cfg.TopBandHeight+middle/2); !sameColour(got, layoutTemplate) {
		t.Errorf("middle of a tall template = %v, want the template colour", got)
	}
	if got := tall.At(5, cfg.TopBandHeight+middle/2); !sameColour(got, layoutBackdrop) {
		t.Errorf("side margin beside a tall template = %v, want the backdrop", got)
	}
	// Nothing of the template may reach into either band.
	if got := tall.At(cfg.Width/2, cfg.TopBandHeight-2); sameColour(got, layoutTemplate) {
		t.Error("template reached into the top band")
	}
	if got := tall.At(cfg.Width/2, cfg.Height-cfg.BottomBandHeight+2); sameColour(got, layoutTemplate) {
		t.Error("template reached into the bottom band")
	}
}

func TestComposeFrame_EmptyCaptionLeavesItsBandBlank(t *testing.T) {
	p := newLayoutProcessor(t)
	cfg := p.config
	out := compose(t, p, 480, 270, "ONLY TOP", "")
	if textPixels(out, 0, cfg.TopBandHeight) == 0 {
		t.Error("top caption missing")
	}
	if got := textPixels(out, cfg.Height-cfg.BottomBandHeight, cfg.Height); got != 0 {
		t.Errorf("%d caption pixels in a bottom band that has no caption", got)
	}
}

func sameColour(c color.Color, want color.RGBA) bool {
	r, g, b, _ := c.RGBA()
	d := func(a uint32, w uint8) bool {
		x := int(a>>8) - int(w)
		return x >= -3 && x <= 3
	}
	return d(r, want.R) && d(g, want.G) && d(b, want.B)
}
