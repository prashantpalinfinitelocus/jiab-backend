package utils

import (
	"image"
	"image/color"
	"testing"
)

// solidImage is a minimal image.Image returning the same color everywhere,
// standing in for a composited meme frame without needing real assets.
type solidImage struct {
	c color.Color
	r image.Rectangle
}

func (s solidImage) ColorModel() color.Model { return color.RGBAModel }
func (s solidImage) Bounds() image.Rectangle { return s.r }
func (s solidImage) At(x, y int) color.Color { return s.c }

// gradientImage produces many distinct colors, standing in for a "busy"
// photographic/video meme template whose content could otherwise dominate a
// pixel-count-weighted vote against a small, flat backdrop sample.
type gradientImage struct {
	r image.Rectangle
}

func (g gradientImage) ColorModel() color.Model { return color.RGBAModel }
func (g gradientImage) Bounds() image.Rectangle { return g.r }
func (g gradientImage) At(x, y int) color.Color {
	return color.RGBA{R: uint8(x * 4 % 256), G: uint8(y * 4 % 256), B: uint8((x + y) * 4 % 256), A: 255}
}

func closestPaletteDistance(pal color.Palette, c color.Color) uint32 {
	best := uint32(1<<32 - 1)
	cr, cg, cb, _ := c.RGBA()
	for _, pc := range pal {
		pr, pg, pb, _ := pc.RGBA()
		dr, dg, db := int64(cr)-int64(pr), int64(cg)-int64(pg), int64(cb)-int64(pb)
		dist := uint32((dr*dr + dg*dg + db*db) >> 16)
		if dist < best {
			best = dist
		}
	}
	return best
}

// TestBuildGlobalPalette_RepresentsAllFrames guards against regressing to the
// old bug: the palette used to come from frame 0 alone, so any later frame
// whose colors weren't in that palette got badly dithered (visible banding,
// "decoloration", per-frame color drift on real GIFs). A palette built across
// every frame should represent a color that appears only in a later frame
// about as well as one from frame 0.
func TestBuildGlobalPalette_RepresentsAllFrames(t *testing.T) {
	p := &ImageProcessor{config: DefaultImageProcessorConfig("", "")}
	bounds := image.Rect(0, 0, 8, 8)

	red := color.RGBA{R: 220, G: 20, B: 20, A: 255}
	blue := color.RGBA{R: 20, G: 20, B: 220, A: 255}
	frames := []image.Image{
		solidImage{c: red, r: bounds},  // frame 0: solid red
		solidImage{c: blue, r: bounds}, // frame 1: solid blue, absent from frame 0
	}

	pal := p.buildGlobalPalette(frames)

	const maxAcceptableDistance = 5000 // squared-distance slack for quantization rounding
	if d := closestPaletteDistance(pal, red); d > maxAcceptableDistance {
		t.Errorf("frame 0's color (red) poorly represented in palette: closest distance %d", d)
	}
	if d := closestPaletteDistance(pal, blue); d > maxAcceptableDistance {
		t.Errorf("frame 1's color (blue) poorly represented in palette: closest distance %d - "+
			"palette likely still only reflects frame 0", d)
	}

	// The forced text/stroke colors must survive untouched so overlay text
	// stays crisp instead of getting dithered.
	if pal[0] != p.config.TextColor {
		t.Errorf("expected palette[0] to be the forced text color, got %v", pal[0])
	}
	if pal[1] != p.config.StrokeColor {
		t.Errorf("expected palette[1] to be the forced stroke color, got %v", pal[1])
	}
}

// TestBuildGlobalPalette_BackdropSurvivesBusyContent guards against
// regressing to the intermediate bug: giving the backdrop only a small
// thumbnail's worth of votes in a single population-weighted quantization
// pass meant a sufficiently colorful meme template could still out-vote it,
// so the backdrop itself started dithering (speckling in otherwise-flat
// regions - reported as the meme's background looking "blurry"/"vibrating").
// A busy, many-colored meme frame must not push the backdrop's own color out
// of the final palette.
func TestBuildGlobalPalette_BackdropSurvivesBusyContent(t *testing.T) {
	backdropColor := color.RGBA{R: 40, G: 160, B: 70, A: 255} // Sprite-green-ish
	p := &ImageProcessor{
		config:   DefaultImageProcessorConfig("", ""),
		backdrop: solidImage{c: backdropColor, r: image.Rect(0, 0, 100, 100)},
	}

	busyFrame := gradientImage{r: image.Rect(0, 0, 64, 64)}
	pal := p.buildGlobalPalette([]image.Image{busyFrame})

	const maxAcceptableDistance = 2000
	if d := closestPaletteDistance(pal, backdropColor); d > maxAcceptableDistance {
		t.Errorf("backdrop color poorly represented against busy content: closest distance %d - "+
			"backdrop is dithering instead of matching exactly", d)
	}
}
