package utils

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/gif"
	"os/exec"
	"testing"
)

// TestEncodeGIFViaFFmpeg_ProducesValidAnimatedGIF exercises the real ffmpeg
// binary (skipped if unavailable, e.g. outside the Docker image) to catch
// mistakes in the concat/filter command construction that a compile-only
// check can't - a malformed filtergraph or concat list still builds fine.
func TestEncodeGIFViaFFmpeg_ProducesValidAnimatedGIF(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed - this test only runs where the real binary is available")
	}

	bounds := image.Rect(0, 0, 16, 16)
	red := color.RGBA{R: 220, G: 20, B: 20, A: 255}
	blue := color.RGBA{R: 20, G: 20, B: 220, A: 255}
	frames := []image.Image{
		solidImage{c: red, r: bounds},
		solidImage{c: blue, r: bounds},
	}
	delays := []int{10, 10}
	pal := color.Palette{red, blue}

	data, err := encodeGIFViaFFmpeg(context.Background(), frames, delays, 0, pal)
	if err != nil {
		t.Fatalf("encodeGIFViaFFmpeg failed: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("encodeGIFViaFFmpeg returned no bytes")
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not a valid GIF: %v", err)
	}
	if len(decoded.Image) < 2 {
		t.Fatalf("expected at least 2 frames in output, got %d", len(decoded.Image))
	}

	if d := closestPaletteDistance(decoded.Image[0].Palette, red); d > 5000 {
		t.Errorf("frame 0 doesn't represent red well: closest distance %d", d)
	}
	if d := closestPaletteDistance(decoded.Image[len(decoded.Image)-1].Palette, blue); d > 5000 {
		t.Errorf("last frame doesn't represent blue well: closest distance %d", d)
	}
}

// TestEncodeGIFViaFFmpeg_UsesSuppliedPaletteNotOwnPalettegen guards against
// the actual production bug: switching to ffmpeg initially still let its
// own palettegen re-derive a palette from the full composited frame
// (backdrop included), silently reintroducing the same population-weighted
// imbalance buildGlobalPalette's reserved backdrop budget exists to prevent
// - a large flat background out-voting busy foreground content for a
// 256-color budget, which showed up as speckling on real memes even after
// switching to ffmpeg. Feeds encodeGIFViaFFmpeg the same
// deliberately-imbalanced scene buildGlobalPalette is tested against
// (TestBuildGlobalPalette_BackdropSurvivesBusyContent) via a palette
// computed the same way, and checks the *encoded output* actually honors
// it - not just that the input palette contained the color.
func TestEncodeGIFViaFFmpeg_UsesSuppliedPaletteNotOwnPalettegen(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not installed - this test only runs where the real binary is available")
	}

	p := &ImageProcessor{config: DefaultImageProcessorConfig("", "")}
	backdropColor := color.RGBA{R: 40, G: 160, B: 70, A: 255}
	p.backdrop = solidImage{c: backdropColor, r: image.Rect(0, 0, 100, 100)}

	// The composited frame: mostly backdrop-green with a small busy patch,
	// mirroring a letterboxed meme on a much larger flat background.
	busyBounds := image.Rect(0, 0, 64, 64)
	pal := p.buildGlobalPalette([]image.Image{gradientImage{r: busyBounds}})

	composited := []image.Image{compositeOverBackdrop{
		backdrop: backdropColor,
		fg:       gradientImage{r: busyBounds},
		fgRect:   image.Rect(4, 4, 60, 60),
		bounds:   image.Rect(0, 0, 64, 64),
	}}

	data, err := encodeGIFViaFFmpeg(context.Background(), composited, []int{10}, 0, pal)
	if err != nil {
		t.Fatalf("encodeGIFViaFFmpeg failed: %v", err)
	}

	decoded, err := gif.DecodeAll(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output is not a valid GIF: %v", err)
	}

	if d := closestPaletteDistance(decoded.Image[0].Palette, backdropColor); d > 2000 {
		t.Errorf("backdrop color poorly represented in ffmpeg output: closest distance %d - "+
			"ffmpeg's own palettegen is likely being used instead of the supplied palette", d)
	}
}

// compositeOverBackdrop draws fg inside fgRect over a flat backdrop color,
// standing in for composeFrame's real output without needing real assets.
type compositeOverBackdrop struct {
	backdrop color.Color
	fg       image.Image
	fgRect   image.Rectangle
	bounds   image.Rectangle
}

func (c compositeOverBackdrop) ColorModel() color.Model { return color.RGBAModel }
func (c compositeOverBackdrop) Bounds() image.Rectangle { return c.bounds }
func (c compositeOverBackdrop) At(x, y int) color.Color {
	pt := image.Pt(x, y)
	if pt.In(c.fgRect) {
		return c.fg.At(x, y)
	}
	return c.backdrop
}
