package utils

import (
	"image"
	"image/color"
	"image/gif"
	"testing"
)

// TestDecodeFullGIFFrames_PartialFrameBoundsPersistPreviousContent guards
// against the actual production bug: GIF encoders routinely only encode the
// region of a frame that changed from the previous one, leaving the rest to
// persist via disposal method DisposalNone. g.Image[i].Bounds() reflects
// that partial rectangle directly - on a real trending Giphy GIF, frames
// after the first started at row 20 or 21, not row 0. Treating that partial
// image as if it were the complete frame (what this code did before)
// silently fed a wrong height into the resize/compositing step, distorting
// nearly every frame differently - reported as "vibrating" when animated and
// speckling/misalignment when viewed as a still.
//
// Builds a 2-frame GIF by hand where frame 1 only covers a small corner
// (mirroring the real asset's partial-frame encoding) and checks the
// reconstructed frame still shows frame 0's color everywhere outside that
// corner, with the new color only inside it.
func TestDecodeFullGIFFrames_PartialFrameBoundsPersistPreviousContent(t *testing.T) {
	const size = 10
	fullPalette := color.Palette{
		color.RGBA{R: 200, G: 20, B: 20, A: 255}, // index 0: frame 0's color
		color.RGBA{R: 20, G: 20, B: 200, A: 255}, // index 1: frame 1's patch color
	}

	frame0 := image.NewPaletted(image.Rect(0, 0, size, size), fullPalette)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			frame0.SetColorIndex(x, y, 0)
		}
	}

	// Frame 1 only covers a small bottom-right patch, like a real optimized
	// GIF encoding just the changed region - not the full canvas.
	patchRect := image.Rect(6, 6, size, size)
	frame1 := image.NewPaletted(patchRect, fullPalette)
	for y := patchRect.Min.Y; y < patchRect.Max.Y; y++ {
		for x := patchRect.Min.X; x < patchRect.Max.X; x++ {
			frame1.SetColorIndex(x, y, 1)
		}
	}

	g := &gif.GIF{
		Image:    []*image.Paletted{frame0, frame1},
		Delay:    []int{10, 10},
		Disposal: []byte{gif.DisposalNone, gif.DisposalNone},
		Config:   image.Config{Width: size, Height: size},
	}

	frames := decodeFullGIFFrames(g)
	if len(frames) != 2 {
		t.Fatalf("expected 2 reconstructed frames, got %d", len(frames))
	}

	full := frames[1]
	if got := full.Bounds(); got != image.Rect(0, 0, size, size) {
		t.Fatalf("reconstructed frame must cover the full canvas, got bounds %v", got)
	}

	// Outside the patch: must still show frame 0's color, not black/empty -
	// this is exactly what broke before (the "missing" region wasn't
	// preserved at all).
	outside := full.At(1, 1)
	if r, g, b, _ := outside.RGBA(); r>>8 != 200 || g>>8 != 20 || b>>8 != 20 {
		t.Errorf("pixel outside the patch should still show frame 0's red (200,20,20), got RGBA %v", outside)
	}

	// Inside the patch: must show frame 1's new color.
	inside := full.At(7, 7)
	if r, g, b, _ := inside.RGBA(); r>>8 != 20 || g>>8 != 20 || b>>8 != 200 {
		t.Errorf("pixel inside the patch should show frame 1's blue (20,20,200), got RGBA %v", inside)
	}
}
