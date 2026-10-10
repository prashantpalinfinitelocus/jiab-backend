package utils

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"io"
	"math"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"

	"github.com/ericpauley/go-quantize/quantize"
	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
	"github.com/sirupsen/logrus"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
)

const (
	FINAL_WIDTH        = 348
	FINAL_HEIGHT       = 348
	FONT_SIZE          = 18.67
	TEXT_TOP_MARGIN    = 32
	TEXT_BOTTOM_MARGIN = 32
	WATERMARK_SCALE    = 0.15
	WATERMARK_MARGIN   = 12
	STROKE_WIDTH       = 0.78
	TEXT_SIDE_PADDING  = 12
)

type ImageProcessorConfig struct {
	Width            int
	Height           int
	FontSize         float64
	TextColor        color.Color
	StrokeColor      color.Color
	StrokeWidth      float64
	WatermarkScale   float64
	WatermarkMargin  int
	TextTopMargin    int
	TextBottomMargin int
	BackdropURL      string
	LogoURL          string
}

func DefaultImageProcessorConfig(backdropURL, logoURL string) *ImageProcessorConfig {
	return &ImageProcessorConfig{
		Width:            FINAL_WIDTH,
		Height:           FINAL_HEIGHT,
		FontSize:         FONT_SIZE,
		TextColor:        color.White,
		StrokeColor:      color.Black,
		StrokeWidth:      STROKE_WIDTH,
		WatermarkScale:   WATERMARK_SCALE,
		WatermarkMargin:  WATERMARK_MARGIN,
		TextTopMargin:    TEXT_TOP_MARGIN,
		TextBottomMargin: TEXT_BOTTOM_MARGIN,
		BackdropURL:      backdropURL,
		LogoURL:          logoURL,
	}
}

type ImageProcessor struct {
	config *ImageProcessorConfig
	font   font.Face
	// Cached static assets
	backdrop   image.Image
	logo       image.Image
	assetsOnce sync.Once
	assetsErr  error
	fontMu     sync.Mutex
}

func NewImageProcessor(config *ImageProcessorConfig, fontPath string) (*ImageProcessor, error) {
	face := loadPreferredFontFace(fontPath, config.FontSize)
	if face == nil {
		return nil, fmt.Errorf("failed to load any usable font face")
	}

	return &ImageProcessor{
		config: config,
		font:   face,
	}, nil
}

// Config returns the immutable config pointer (read-only usage expected)
func (p *ImageProcessor) Config() *ImageProcessorConfig {
	return p.config
}

func loadPreferredFontFace(primaryPath string, size float64) font.Face {
	// Preferred bold faces ordered by desirability
	candidates := []string{}
	if primaryPath != "" {
		candidates = append(candidates, primaryPath)
	}
	candidates = append(candidates,
		// Alpine's font-dejavu package (our actual runtime image) installs
		// here, not under a "truetype" subdirectory like Debian/Ubuntu -
		// this was missing entirely, so every deployment silently fell
		// through this whole list to basicfont.Face7x13.
		"/usr/share/fonts/dejavu/DejaVuSans-Bold.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		// Common Debian/Ubuntu paths
		"/usr/share/fonts/truetype/impact/Impact.ttf",
		"/usr/share/fonts/truetype/msttcorefonts/Impact.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSansNarrow-Bold.ttf",
		"/usr/share/fonts/truetype/freefont/FreeSansBold.ttf",
		// macOS
		"/Library/Fonts/Impact.ttf",
		"/System/Library/Fonts/Supplemental/Arial Black.ttf",
		"/System/Library/Fonts/Supplemental/Arial Bold.ttf",
		// Fallback regular that still looks solid
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
	)

	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if f, err := gg.LoadFontFace(p, size); err == nil {
				return f
			}
		}
	}
	// Final fallback to basic bitmap font
	return basicfont.Face7x13
}

func (p *ImageProcessor) ProcessStaticImage(ctx context.Context, memeFile *multipart.FileHeader, topText, bottomText string) (image.Image, error) {
	memeImg, err := decodeImageFromFile(memeFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode meme image: %w", err)
	}

	if err := p.ensureAssets(ctx); err != nil {
		return nil, err
	}

	return p.composeFrame(memeImg, p.backdrop, p.logo, topText, bottomText), nil
}

// ProcessGIF composites the text/logo overlay onto every frame of the source
// GIF and returns fully-encoded GIF bytes.
//
// The palette itself always comes from buildGlobalPalette - reserving a
// fixed budget for the backdrop/logo instead of letting them compete with
// the meme's own content by pixel count is what actually fixed the
// backdrop-speckling bug, and letting ffmpeg's own palettegen re-derive a
// palette from the full composited frame (backdrop included) would silently
// undo that fix by reintroducing the exact same population-weighted
// imbalance. ffmpeg is used only for the dithering step (paletteuse), where
// its algorithms (e.g. sierra2_4a) are more refined than Go's Floyd-
// Steinberg, when the ffmpeg binary is present in the runtime image. Falls
// back to a pure Go-native dither when it isn't (e.g. local `go run` outside
// the container), so this never hard-fails on it being absent.
func (p *ImageProcessor) ProcessGIF(ctx context.Context, gifFile *multipart.FileHeader, topText, bottomText string) ([]byte, error) {
	file, err := gifFile.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open gif file: %w", err)
	}
	defer file.Close()

	originalGIF, err := gif.DecodeAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode gif: %w", err)
	}

	if err := p.ensureAssets(ctx); err != nil {
		return nil, err
	}

	// GIF encoders routinely only encode the region of each frame that
	// actually changed from the previous one (a standard size optimization),
	// leaving the rest to persist via the frame's disposal method - which is
	// exactly what originalGIF.Image[i] is: each frame's own raw bounds,
	// frequently a partial rectangle (e.g. (0,21)-(480,480), missing the top
	// rows entirely) rather than the full canvas. Treating that partial
	// image as a complete frame - what this did before - fed containImage
	// the wrong height, stretching/misaligning the visible content
	// differently on nearly every frame. Viewed as a still it looked like
	// scattered noise at edges; animated, it's what was reported as the
	// image "vibrating". fullFrames reconstructs each frame's true,
	// complete appearance by compositing according to GIF disposal
	// semantics before anything else touches it.
	fullFrames := decodeFullGIFFrames(originalGIF)

	frameCount := len(fullFrames)
	composited := make([]image.Image, frameCount)
	delays := make([]int, frameCount)

	// Process frames concurrently with a bounded worker pool
	workerCount := runtime.NumCPU()
	sem := make(chan struct{}, workerCount)
	var wg sync.WaitGroup

	for i, frame := range fullFrames {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, frm image.Image) {
			defer wg.Done()
			defer func() { <-sem }()
			composited[idx] = p.composeFrame(frm, p.backdrop, p.logo, topText, bottomText)
			if idx < len(originalGIF.Delay) {
				delays[idx] = originalGIF.Delay[idx]
			} else {
				delays[idx] = 10
			}
		}(i, frame)
	}
	wg.Wait()

	// Quantize once across the whole animation, not just frame 0: a palette
	// sized for a single frame routinely doesn't fit the rest of the clip,
	// which is what produced the banding/posterization and per-frame color
	// drift ("fluctuating", "decoloration") seen on photographic source GIFs.
	// Sourced from the meme's own fully-reconstructed frames rather than the
	// composited ones - see buildGlobalPalette for why.
	sourceImages := make([]image.Image, frameCount)
	for i, frame := range fullFrames {
		sourceImages[i] = frame
	}
	globalPal := p.buildGlobalPalette(sourceImages)

	if ffmpegPath, lookErr := exec.LookPath("ffmpeg"); lookErr == nil {
		data, err := encodeGIFViaFFmpeg(ctx, composited, delays, originalGIF.LoopCount, globalPal)
		if err == nil {
			return data, nil
		}
		// Fall through to the Go-native path below rather than failing the
		// request outright over an ffmpeg-side problem - but this must not
		// fail silently: it's exactly what made a previous deployment look
		// unfixed even though the ffmpeg code path had shipped, with nothing
		// in the response or logs to tell the two apart.
		logrus.WithError(err).WithField("ffmpeg_path", ffmpegPath).
			Warn("ffmpeg GIF encoding failed, falling back to Go-native dither")
	} else {
		logrus.WithError(lookErr).Warn("ffmpeg not found on PATH, using Go-native GIF dither")
	}

	return encodeGIFGoNative(composited, globalPal, delays, originalGIF.LoopCount, originalGIF.Disposal)
}

// decodeFullGIFFrames reconstructs each frame's complete, true appearance
// from a decoded GIF, honoring disposal methods - g.Image[i] alone is
// frequently only the sub-rectangle that changed from the previous frame,
// not the full canvas (see the comment in ProcessGIF for why that matters).
// This is the same reconstruction any GIF viewer/browser performs
// internally; the stdlib decoder deliberately leaves it to the caller.
func decodeFullGIFFrames(g *gif.GIF) []*image.RGBA {
	screen := image.Rect(0, 0, g.Config.Width, g.Config.Height)
	canvas := image.NewRGBA(screen)

	frames := make([]*image.RGBA, len(g.Image))
	var previousSnapshot *image.RGBA
	var previousRect image.Rectangle
	var previousDisposal byte

	for i, frame := range g.Image {
		if i > 0 {
			switch previousDisposal {
			case gif.DisposalBackground:
				draw.Draw(canvas, previousRect, image.NewUniform(color.Transparent), image.Point{}, draw.Src)
			case gif.DisposalPrevious:
				if previousSnapshot != nil {
					draw.Draw(canvas, previousRect, previousSnapshot, previousRect.Min, draw.Src)
				}
			}
		}

		disposal := byte(gif.DisposalNone)
		if i < len(g.Disposal) {
			disposal = g.Disposal[i]
		}
		if disposal == gif.DisposalPrevious {
			snap := image.NewRGBA(frame.Bounds())
			draw.Draw(snap, frame.Bounds(), canvas, frame.Bounds().Min, draw.Src)
			previousSnapshot = snap
		} else {
			previousSnapshot = nil
		}

		// frame's own palette already encodes its transparency index as an
		// RGBA entry with alpha 0 (Go's gif decoder resolves that for us),
		// so a normal alpha-composite here correctly lets the canvas below
		// show through the frame's transparent pixels.
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)

		full := image.NewRGBA(screen)
		draw.Draw(full, screen, canvas, image.Point{}, draw.Src)
		frames[i] = full

		previousRect = frame.Bounds()
		previousDisposal = disposal
	}
	return frames
}

// encodeGIFGoNative is the fallback path for environments without ffmpeg.
func encodeGIFGoNative(composited []image.Image, globalPal color.Palette, delays []int, loopCount int, disposal []byte) ([]byte, error) {
	frameCount := len(composited)
	processedFrames := make([]*image.Paletted, frameCount)
	workerCount := runtime.NumCPU()
	sem := make(chan struct{}, workerCount)
	var wg sync.WaitGroup
	for i, compositeImg := range composited {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, img image.Image) {
			defer wg.Done()
			defer func() { <-sem }()
			palettedFrame := image.NewPaletted(img.Bounds(), globalPal)
			// Dither into the shared global palette to preserve perceived detail
			draw.FloydSteinberg.Draw(palettedFrame, palettedFrame.Bounds(), img, image.Point{})
			processedFrames[idx] = palettedFrame
		}(i, compositeImg)
	}
	wg.Wait()

	g := &gif.GIF{
		Image:     processedFrames,
		Delay:     delays,
		LoopCount: loopCount,
		Disposal:  disposal,
		Config: image.Config{
			ColorModel: globalPal,
			Width:      processedFrames[0].Bounds().Dx(),
			Height:     processedFrames[0].Bounds().Dy(),
		},
	}

	var buf bytes.Buffer
	if err := EncodeGIFToWriter(g, &buf); err != nil {
		return nil, fmt.Errorf("failed to encode gif: %w", err)
	}
	return buf.Bytes(), nil
}

// buildGlobalPalette computes one <=256-color palette that fits the whole
// animation via median-cut quantization, forcing the overlay's text/stroke
// colors to survive exactly (unquantized, so the caption stays crisp instead
// of dithered).
//
// The meme content, backdrop, and logo each get their own reserved share of
// the palette via separate quantization passes, rather than pooling all
// their pixels into one vote weighted by pixel count. That population-
// weighted approach couldn't win either way: feeding the full composite let
// the backdrop's huge flat area dominate and starve the meme's own line art
// of distinct shades (dithering noise at the meme's edges); down-weighting
// the backdrop to a small thumbnail to fix that then gave median-cut too few
// backdrop votes to bother allocating it an exact slot, so the backdrop
// itself started dithering instead (speckling in otherwise-flat regions,
// varying with how much color budget a given template's content demanded).
// Reserving fixed budgets sidesteps the tradeoff: the backdrop and logo
// dither with ~zero error against their own dedicated colors, and the
// meme still gets the large majority of the palette for its own content.
func (p *ImageProcessor) buildGlobalPalette(memeFrames []image.Image) color.Palette {
	const maxSampleFrames = 24
	sample := memeFrames
	if len(memeFrames) > maxSampleFrames {
		sample = make([]image.Image, maxSampleFrames)
		for i := range sample {
			sample[i] = memeFrames[i*len(memeFrames)/maxSampleFrames]
		}
	}

	quantizer := quantize.MedianCutQuantizer{}
	pal := color.Palette{p.config.TextColor, p.config.StrokeColor}

	const backdropBudget = 24
	const logoBudget = 8
	const quantizeSampleSize = 64

	if p.backdrop != nil {
		withBudget := make(color.Palette, len(pal), len(pal)+backdropBudget)
		copy(withBudget, pal)
		backdropSample := resize.Resize(quantizeSampleSize, quantizeSampleSize, p.backdrop, resize.Lanczos3)
		pal = quantizer.Quantize(withBudget, backdropSample)
	}
	if p.logo != nil {
		withBudget := make(color.Palette, len(pal), len(pal)+logoBudget)
		copy(withBudget, pal)
		logoSample := resize.Resize(quantizeSampleSize, quantizeSampleSize, p.logo, resize.Lanczos3)
		pal = quantizer.Quantize(withBudget, logoSample)
	}

	finalPal := make(color.Palette, len(pal), 256)
	copy(finalPal, pal)
	return quantizer.Quantize(finalPal, stackImagesVertically(sample))
}

// stackImagesVertically concatenates images of possibly different sizes into
// one tall image, so the quantizer can analyze color usage across all of
// them - meme frames and asset thumbnails alike - in a single pass.
func stackImagesVertically(images []image.Image) image.Image {
	maxWidth := 0
	totalHeight := 0
	for _, img := range images {
		b := img.Bounds()
		if b.Dx() > maxWidth {
			maxWidth = b.Dx()
		}
		totalHeight += b.Dy()
	}

	canvas := image.NewRGBA(image.Rect(0, 0, maxWidth, totalHeight))
	y := 0
	for _, img := range images {
		b := img.Bounds()
		dstRect := image.Rect(0, y, b.Dx(), y+b.Dy())
		draw.Draw(canvas, dstRect, img, b.Min, draw.Src)
		y += b.Dy()
	}
	return canvas
}

func (p *ImageProcessor) composeFrame(memeImg, backdropImg, logoImg image.Image, topText, bottomText string) image.Image {
	dc := gg.NewContext(p.config.Width, p.config.Height)

	backdropResized := resize.Resize(uint(p.config.Width), uint(p.config.Height), backdropImg, resize.Lanczos3)
	dc.DrawImage(backdropResized, 0, 0)

	memeContained := p.containImage(memeImg, p.config.Width, p.config.Height)
	offsetX := (p.config.Width - memeContained.Bounds().Dx()) / 2
	offsetY := (p.config.Height - memeContained.Bounds().Dy()) / 2
	dc.DrawImage(memeContained, offsetX, offsetY)

	dc.SetFontFace(p.font)
	dc.SetColor(p.config.TextColor)

	if topText != "" {
		p.drawWrappedTopText(dc, strings.ToUpper(topText))
	}

	if bottomText != "" {
		p.drawWrappedBottomText(dc, strings.ToUpper(bottomText))
	}

	logoResized := p.resizeLogo(logoImg)
	logoX := p.config.Width - logoResized.Bounds().Dx() - p.config.WatermarkMargin
	logoY := p.config.Height - logoResized.Bounds().Dy() - p.config.WatermarkMargin
	dc.DrawImage(logoResized, logoX, logoY)

	return dc.Image()
}

func (p *ImageProcessor) drawTextLineWithStroke(dc *gg.Context, text string, x, y float64, vAnchor float64) {
	p.fontMu.Lock()
	defer p.fontMu.Unlock()
	// Make outline thickness proportional to font size if larger than configured
	stroke := p.config.StrokeWidth
	if dynamic := p.config.FontSize * 0.11; dynamic > stroke {
		stroke = dynamic
	}

	dc.SetColor(p.config.StrokeColor)
	dc.SetLineWidth(stroke)
	dc.SetLineJoinRound()

	for dx := -1.0; dx <= 1.0; dx++ {
		for dy := -1.0; dy <= 1.0; dy++ {
			if dx != 0 || dy != 0 {
				dc.DrawStringAnchored(text, x+dx*stroke/2, y+dy*stroke/2, 0.5, vAnchor)
			}
		}
	}

	// Slight multi-pass fill to simulate extra-bold weight when font face isn't bold enough
	dc.SetColor(p.config.TextColor)
	dc.DrawStringAnchored(text, x, y, 0.5, vAnchor)
	dc.DrawStringAnchored(text, x+0.25, y, 0.5, vAnchor)
	dc.DrawStringAnchored(text, x-0.25, y, 0.5, vAnchor)
}

func (p *ImageProcessor) drawWrappedTopText(dc *gg.Context, text string) {
	maxWidth := float64(p.config.Width - 2*TEXT_SIDE_PADDING)
	lines := p.wrapText(dc, text, maxWidth)
	// A real font's rendered glyph height (ascent + descent) typically runs
	// well past its nominal "font size" - stacking lines at exactly 100% of
	// that size crowds one line's descenders into the next line's ascenders,
	// which read as the caption's letters running into each other. 1.2x is
	// the standard typographic leading for this reason.
	lineHeight := p.config.FontSize * 1.2
	startY := float64(p.config.TextTopMargin)

	x := float64(p.config.Width) / 2
	y := startY
	for _, line := range lines {
		p.drawTextLineWithStroke(dc, line, x, y, 0.0)
		y += lineHeight
	}
}

func (p *ImageProcessor) drawWrappedBottomText(dc *gg.Context, text string) {
	maxWidth := float64(p.config.Width - 2*TEXT_SIDE_PADDING)
	lines := p.wrapText(dc, text, maxWidth)
	// A real font's rendered glyph height (ascent + descent) typically runs
	// well past its nominal "font size" - stacking lines at exactly 100% of
	// that size crowds one line's descenders into the next line's ascenders,
	// which read as the caption's letters running into each other. 1.2x is
	// the standard typographic leading for this reason.
	lineHeight := p.config.FontSize * 1.2
	x := float64(p.config.Width) / 2
	// Anchor the block exactly p.config.TextBottomMargin from the bottom edge
	y := float64(p.config.Height - p.config.TextBottomMargin)
	for i := len(lines) - 1; i >= 0; i-- {
		p.drawTextLineWithStroke(dc, lines[i], x, y, 1.0)
		y -= lineHeight
	}
}

func (p *ImageProcessor) wrapText(dc *gg.Context, text string, maxWidth float64) []string {
	p.fontMu.Lock()
	defer p.fontMu.Unlock()
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var current string

	for _, w := range words {
		candidate := w
		if current != "" {
			candidate = current + " " + w
		}
		width, _ := dc.MeasureString(candidate)
		if width <= maxWidth {
			current = candidate
			continue
		}

		if current == "" {
			// Single very long word: hard-break
			current = p.hardBreakWord(dc, w, maxWidth)
			lines = append(lines, current)
			current = ""
			continue
		}

		lines = append(lines, current)
		current = w
	}

	if current != "" {
		lines = append(lines, current)
	}

	return lines
}

func (p *ImageProcessor) hardBreakWord(dc *gg.Context, word string, maxWidth float64) string {
	// Best-effort hard break: reduce word until it fits
	for i := len(word); i > 1; i-- {
		w := word[:i]
		if width, _ := dc.MeasureString(w); width <= maxWidth {
			return w
		}
	}
	return word
}

func (p *ImageProcessor) containImage(img image.Image, targetWidth, targetHeight int) image.Image {
	bounds := img.Bounds()
	imgWidth := bounds.Dx()
	imgHeight := bounds.Dy()

	scaleW := float64(targetWidth) / float64(imgWidth)
	scaleH := float64(targetHeight) / float64(imgHeight)
	scale := math.Min(scaleW, scaleH)

	newWidth := uint(math.Round(float64(imgWidth) * scale))
	newHeight := uint(math.Round(float64(imgHeight) * scale))

	return resize.Resize(newWidth, newHeight, img, resize.Lanczos3)
}

func (p *ImageProcessor) resizeLogo(logo image.Image) image.Image {
	bounds := logo.Bounds()
	logoWidth := bounds.Dx()
	logoHeight := bounds.Dy()

	targetWidth := int(float64(p.config.Width) * p.config.WatermarkScale)
	aspect := float64(logoHeight) / float64(logoWidth)
	targetHeight := int(float64(targetWidth) * aspect)

	return resize.Resize(uint(targetWidth), uint(targetHeight), logo, resize.Lanczos3)
}

func decodeImageFromFile(fileHeader *multipart.FileHeader) (image.Image, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	return img, nil
}

func downloadImageFromURL(ctx context.Context, url string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request for %s: %w", url, err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download image from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download image from %s: status code %d", url, resp.StatusCode)
	}

	img, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image from %s: %w", url, err)
	}

	return img, nil
}

// ensureAssets downloads and caches backdrop and logo once per process
func (p *ImageProcessor) ensureAssets(ctx context.Context) error {
	p.assetsOnce.Do(func() {
		var wg sync.WaitGroup
		wg.Add(2)

		var backdrop image.Image
		var logo image.Image
		var err1 error
		var err2 error

		go func() {
			defer wg.Done()
			backdrop, err1 = downloadImageFromURL(ctx, p.config.BackdropURL)
		}()

		go func() {
			defer wg.Done()
			logo, err2 = downloadImageFromURL(ctx, p.config.LogoURL)
		}()

		wg.Wait()
		if err1 != nil {
			p.assetsErr = fmt.Errorf("failed to download backdrop image: %w", err1)
			return
		}
		if err2 != nil {
			p.assetsErr = fmt.Errorf("failed to download logo image: %w", err2)
			return
		}
		p.backdrop = backdrop
		p.logo = logo
	})

	return p.assetsErr
}


func EncodeImageToWriter(img image.Image, w io.Writer) error {
	return png.Encode(w, img)
}

func EncodeGIFToWriter(g *gif.GIF, w io.Writer) error {
	// Preserve disposal and loop count; avoid re-quantization artifacts
	return gif.EncodeAll(w, &gif.GIF{
		Image:           g.Image,
		Delay:           g.Delay,
		LoopCount:       g.LoopCount,
		Disposal:        g.Disposal,
		Config:          g.Config,
		BackgroundIndex: g.BackgroundIndex,
	})
}
