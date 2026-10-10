package utils

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const ffmpegEncodeTimeout = 25 * time.Second

// encodeGIFViaFFmpeg encodes composited frames into a GIF using the supplied
// palette (from buildGlobalPalette) and ffmpeg's paletteuse filter for
// dithering. The palette is precomputed rather than left to ffmpeg's own
// palettegen deliberately: palettegen analyzing the full composited frame
// (backdrop included) reintroduces the exact population-weighted imbalance
// buildGlobalPalette's reserved backdrop/logo budget was built to avoid - a
// large flat background otherwise dominates the vote and starves the meme's
// own content of distinct shades. ffmpeg's role here is purely the dithering
// step, where its algorithms are more refined than Go's Floyd-Steinberg.
func encodeGIFViaFFmpeg(ctx context.Context, frames []image.Image, delays []int, loopCount int, pal color.Palette) ([]byte, error) {
	if len(frames) == 0 {
		return nil, fmt.Errorf("no frames to encode")
	}

	tempDir, err := os.MkdirTemp("", "meme-gif-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	concatListPath := filepath.Join(tempDir, "frames.txt")
	if err := writeFramesAndConcatList(tempDir, concatListPath, frames, delays); err != nil {
		return nil, err
	}

	palettePath := filepath.Join(tempDir, "palette.png")
	if err := writePaletteImage(palettePath, pal); err != nil {
		return nil, err
	}

	outputPath := filepath.Join(tempDir, "output.gif")
	ctx, cancel := context.WithTimeout(ctx, ffmpegEncodeTimeout)
	defer cancel()

	loop := "0" // ffmpeg: 0 = infinite loop, matching GIF's own LoopCount==0 convention
	if loopCount != 0 {
		loop = fmt.Sprintf("%d", loopCount)
	}

	// #nosec G204 -- binary name is the fixed literal "ffmpeg"; concatListPath/
	// palettePath/outputPath are all filepath.Join(tempDir, <fixed name>) off
	// an os.MkdirTemp directory (see caller), never user input.
	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-y",
		"-f", "concat",
		"-safe", "0",
		"-i", concatListPath,
		"-i", palettePath,
		"-lavfi", "paletteuse=dither=sierra2_4a",
		"-loop", loop,
		outputPath,
	)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg failed: %w: %s", err, stderr.String())
	}

	return os.ReadFile(outputPath) // #nosec G304 -- see tempDir note above
}

// writePaletteImage writes pal as a 16x16 PNG in the layout ffmpeg's own
// palettegen produces (row-major, one color per pixel), which paletteuse
// reads back as its "-i palette.png" input regardless of which tool
// generated it. Unused cells beyond len(pal) are left fully transparent.
func writePaletteImage(path string, pal color.Palette) error {
	const gridSize = 16 // 16x16 = 256 cells, GIF's max palette size
	img := image.NewRGBA(image.Rect(0, 0, gridSize, gridSize))
	for i := 0; i < gridSize*gridSize; i++ {
		x, y := i%gridSize, i/gridSize
		if i < len(pal) {
			img.Set(x, y, pal[i])
		} else {
			img.Set(x, y, color.RGBA{})
		}
	}

	f, err := os.Create(path) // #nosec G304 -- caller passes tempDir/"palette.png" (os.MkdirTemp), never user input
	if err != nil {
		return fmt.Errorf("failed to create palette file: %w", err)
	}
	err = EncodeImageToWriter(img, f)
	closeErr := f.Close()
	if err != nil {
		return fmt.Errorf("failed to encode palette image: %w", err)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close palette file: %w", closeErr)
	}
	return nil
}

// writeFramesAndConcatList writes each frame as a PNG and a concat-demuxer
// list file pairing it with its delay (converted from GIF's centiseconds to
// seconds). The concat demuxer only honors a "duration" line if the file it
// applies to is followed by another entry, so the last frame is listed twice
// - a well-documented quirk of the demuxer, not a mistake.
func writeFramesAndConcatList(tempDir, concatListPath string, frames []image.Image, delays []int) error {
	var list strings.Builder
	for i, frame := range frames {
		framePath := filepath.Join(tempDir, fmt.Sprintf("frame_%04d.png", i))
		f, err := os.Create(framePath) // #nosec G304 -- tempDir is from os.MkdirTemp, frame_%04d.png is a fixed pattern, never user input
		if err != nil {
			return fmt.Errorf("failed to create frame file: %w", err)
		}
		err = EncodeImageToWriter(frame, f)
		closeErr := f.Close()
		if err != nil {
			return fmt.Errorf("failed to encode frame %d: %w", i, err)
		}
		if closeErr != nil {
			return fmt.Errorf("failed to close frame %d: %w", i, closeErr)
		}

		delayCentiseconds := delays[i]
		if delayCentiseconds <= 0 {
			delayCentiseconds = 10
		}
		durationSeconds := float64(delayCentiseconds) / 100.0

		fmt.Fprintf(&list, "file '%s'\n", framePath)
		fmt.Fprintf(&list, "duration %f\n", durationSeconds)
	}
	// Duplicate the last frame's entry (without a duration line) to work
	// around the concat demuxer dropping the final duration otherwise.
	lastFramePath := filepath.Join(tempDir, fmt.Sprintf("frame_%04d.png", len(frames)-1))
	fmt.Fprintf(&list, "file '%s'\n", lastFramePath)

	return os.WriteFile(concatListPath, []byte(list.String()), 0o600)
}
