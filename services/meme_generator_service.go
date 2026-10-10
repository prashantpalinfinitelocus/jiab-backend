package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	"github.com/Infinite-Locus-Product/coca_cola_backend/constants"
	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/google/uuid"
)

const (
	CONTENT_TYPE_GIF = "image/gif"
	CONTENT_TYPE_PNG = "image/png"
)

type MemeGeneratorService interface {
	GenerateMeme(ctx context.Context, userID string, memeFile *multipart.FileHeader, topText, bottomText string) (string, error)
	GenerateMemeInMemory(ctx context.Context, memeFile *multipart.FileHeader, topText, bottomText string) ([]byte, string, error)
}

type memeGeneratorService struct {
	imageProcessor *utils.ImageProcessor
	gcsService     utils.GCSService
	pythonURL      string
}

func NewMemeGeneratorService(gcsService utils.GCSService, cfg *config.Config) (MemeGeneratorService, error) {
	fontPath := cfg.MemeGeneratorConfig.FontPath
	if fontPath == "" {
		fontPath = "/app/fonts/font.ttf"
	}

	backdropURL := cfg.MemeGeneratorConfig.BackdropURL
	if backdropURL == "" {
		backdropURL = constants.MEME_BACKDROP_URL
	}

	logoURL := cfg.MemeGeneratorConfig.LogoURL
	if logoURL == "" {
		logoURL = constants.MEME_LOGO_URL
	}

	processorConfig := utils.DefaultImageProcessorConfig(backdropURL, logoURL)
	processor, err := utils.NewImageProcessor(processorConfig, fontPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize image processor: %w", err)
	}

	return &memeGeneratorService{
		imageProcessor: processor,
		gcsService:     gcsService,
		pythonURL:      cfg.MemeGeneratorConfig.PythonServiceURL,
	}, nil
}

func (s *memeGeneratorService) GenerateMeme(ctx context.Context, userID string, memeFile *multipart.FileHeader, topText, bottomText string) (string, error) {
	if memeFile == nil {
		return "", errs.NewBadRequestError("meme file is required", nil)
	}

	isGIF := s.isGIFFile(memeFile)
	var outputURL string
	var err error

	if isGIF {
		outputURL, err = s.processAndUploadGIF(ctx, userID, memeFile, topText, bottomText)
	} else {
		outputURL, err = s.processAndUploadStatic(ctx, userID, memeFile, topText, bottomText)
	}

	if err != nil {
		return "", err
	}

	return outputURL, nil
}

func (s *memeGeneratorService) GenerateMemeInMemory(ctx context.Context, memeFile *multipart.FileHeader, topText, bottomText string) ([]byte, string, error) {
	if memeFile == nil {
		return nil, "", errs.NewBadRequestError("meme file is required", nil)
	}

	isGIF := s.isGIFFile(memeFile)

	if isGIF {
		data, err := s.imageProcessor.ProcessGIF(ctx, memeFile, topText, bottomText)
		if err != nil {
			return nil, "", err
		}

		return data, CONTENT_TYPE_GIF, nil
	}

	processedImg, err := s.imageProcessor.ProcessStaticImage(ctx, memeFile, topText, bottomText)
	if err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	if err := utils.EncodeImageToWriter(processedImg, &buf); err != nil {
		return nil, "", err
	}

	return buf.Bytes(), CONTENT_TYPE_PNG, nil
}

func (s *memeGeneratorService) processAndUploadStatic(ctx context.Context, userID string, memeFile *multipart.FileHeader, topText, bottomText string) (string, error) {
	processedImg, err := s.imageProcessor.ProcessStaticImage(ctx, memeFile, topText, bottomText)
	if err != nil {
		return "", err
	}

	return s.uploadProcessedImage(ctx, userID, processedImg)
}

func (s *memeGeneratorService) processAndUploadGIF(ctx context.Context, userID string, gifFile *multipart.FileHeader, topText, bottomText string) (string, error) {
	// Prefer Python service if configured
	if s.pythonURL != "" {
		data, contentType, err := s.callPythonGIFProcessor(ctx, gifFile, topText, bottomText)
		if err == nil && len(data) > 0 && contentType == CONTENT_TYPE_GIF {
			return s.uploadGIFBytes(ctx, userID, data)
		}
		// fallback to Go path if Python call fails
	}

	data, err := s.imageProcessor.ProcessGIF(ctx, gifFile, topText, bottomText)
	if err != nil {
		return "", err
	}

	return s.uploadGIFBytes(ctx, userID, data)
}

func (s *memeGeneratorService) callPythonGIFProcessor(ctx context.Context, gifFile *multipart.FileHeader, topText, bottomText string) ([]byte, string, error) {
	pyURL := strings.TrimRight(s.pythonURL, "/") + "/process"
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// meme file
	fileWriter, err := writer.CreateFormFile("meme", gifFile.Filename)
	if err != nil {
		return nil, "", err
	}
	f, err := gifFile.Open()
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	if _, err := io.Copy(fileWriter, f); err != nil {
		return nil, "", err
	}

	// text fields
	_ = writer.WriteField("top_text", topText)
	_ = writer.WriteField("bottom_text", bottomText)
	_ = writer.WriteField("backdrop_url", s.imageProcessor.Config().BackdropURL)
	_ = writer.WriteField("logo_url", s.imageProcessor.Config().LogoURL)
	// optional font path from config
	// leave empty to use service defaults

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, pyURL, body)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	// basic URL validation
	if _, err := url.ParseRequestURI(pyURL); err != nil {
		return nil, "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("python service error: %s", resp.Status)
	}

	out, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	return out, CONTENT_TYPE_GIF, nil
}

func (s *memeGeneratorService) uploadGIFBytes(ctx context.Context, userID string, data []byte) (string, error) {
	randomID := uuid.New().String()
	filePath := fmt.Sprintf("%s/%s/%s.gif", constants.MEME_CONTENT_FOLDER, userID, randomID)
	_, gcsPath, err := s.gcsService.UploadFileFromBytes(ctx, data, filePath, CONTENT_TYPE_GIF)
	if err != nil {
		return "", fmt.Errorf("failed to upload to GCS: %w", err)
	}
	signedURL, err := s.gcsService.GetFileSignedURL(ctx, gcsPath, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}
	return signedURL, nil
}

func (s *memeGeneratorService) uploadProcessedImage(ctx context.Context, userID string, img image.Image) (string, error) {
	var buf bytes.Buffer
	if err := utils.EncodeImageToWriter(img, &buf); err != nil {
		return "", fmt.Errorf("failed to encode image: %w", err)
	}

	randomID := uuid.New().String()
	filePath := fmt.Sprintf("%s/%s/%s.png", constants.MEME_CONTENT_FOLDER, userID, randomID)

	_, gcsPath, err := s.gcsService.UploadFileFromBytes(ctx, buf.Bytes(), filePath, CONTENT_TYPE_PNG)
	if err != nil {
		return "", fmt.Errorf("failed to upload to GCS: %w", err)
	}

	signedURL, err := s.gcsService.GetFileSignedURL(ctx, gcsPath, 24*time.Hour)
	if err != nil {
		return "", fmt.Errorf("failed to generate signed URL: %w", err)
	}

	return signedURL, nil
}

func (s *memeGeneratorService) isGIFFile(fileHeader *multipart.FileHeader) bool {
	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	return ext == ".gif"
}
