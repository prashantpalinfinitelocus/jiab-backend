package utils

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	errs "github.com/Infinite-Locus-Product/coca_cola_backend/errors"
)

const (
	remoteAssetFetchTimeout = 15 * time.Second
	// Generous ceiling for a meme template (GIF/PNG/JPEG); bounds memory use
	// against a malicious or misbehaving allowed host.
	remoteAssetMaxBytes = 25 * 1024 * 1024
)

// remoteAssetAllowedHost is the only host FetchRemoteAssetAsFileHeader may
// fetch from - this fetches a client-supplied URL from the server, so an
// unrestricted allowlist would make this an open SSRF proxy (e.g. reaching
// cloud metadata endpoints or internal services). Mirrors the frontend's own
// allowlist in src/app/api/proxy-image/route.ts; widen only with equal
// scrutiny, and only by adding another literal `hostname != "..."` arm below
// (not a map/slice lookup) - CodeQL's request-forgery barrier-guard analysis
// only recognizes a direct equality comparison against a string literal at
// the call site, not a lookup against a variable allowlist.
const remoteAssetAllowedHost = "storage.googleapis.com"

// FetchRemoteAssetAsFileHeader fetches rawURL server-side and wraps the bytes
// in a *multipart.FileHeader, so callers written against a multipart upload
// (image_processor.go's ProcessGIF/ProcessStaticImage) can accept a
// server-fetched template exactly like a client-uploaded one.
//
// rawURL must be https and its host must be remoteAssetAllowedHost.
func FetchRemoteAssetAsFileHeader(rawURL string) (*multipart.FileHeader, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return nil, errs.NewBadRequestError("invalid meme_url", err)
	}
	if parsed.Scheme != "https" {
		return nil, errs.NewBadRequestError("meme_url must use https", nil)
	}
	if parsed.Hostname() != remoteAssetAllowedHost {
		return nil, errs.NewBadRequestError("meme_url host is not allowed", nil)
	}

	client := &http.Client{Timeout: remoteAssetFetchTimeout}
	resp, err := client.Get(parsed.String())
	if err != nil {
		return nil, errs.NewBadRequestError("failed to fetch meme_url", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errs.NewBadRequestError(fmt.Sprintf("failed to fetch meme_url: status %d", resp.StatusCode), nil)
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		return nil, errs.NewBadRequestError("meme_url did not return an image", nil)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, remoteAssetMaxBytes+1))
	if err != nil {
		return nil, errs.NewBadRequestError("failed to read meme_url response", err)
	}
	if len(data) > remoteAssetMaxBytes {
		return nil, errs.NewBadRequestError("meme_url asset is too large", nil)
	}

	filename := path.Base(parsed.Path)
	if filename == "" || filename == "." || filename == "/" {
		filename = "template"
	}
	if !strings.Contains(filename, ".") {
		ext := ".jpg"
		switch {
		case strings.Contains(contentType, "gif"):
			ext = ".gif"
		case strings.Contains(contentType, "png"):
			ext = ".png"
		case strings.Contains(contentType, "webp"):
			ext = ".webp"
		}
		filename += ext
	}

	return buildFileHeader(data, filename, contentType)
}

// buildFileHeader wraps raw bytes in a real *multipart.FileHeader by
// round-tripping them through an in-memory multipart form, since
// multipart.FileHeader has no public constructor.
func buildFileHeader(data []byte, filename, contentType string) (*multipart.FileHeader, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	partHeader := make(map[string][]string)
	partHeader["Content-Disposition"] = []string{fmt.Sprintf(`form-data; name="file"; filename=%q`, filename)}
	if contentType != "" {
		partHeader["Content-Type"] = []string{contentType}
	}
	part, err := writer.CreatePart(partHeader)
	if err != nil {
		return nil, errs.NewInternalServerError("failed to build in-memory upload", err)
	}
	if _, err := part.Write(data); err != nil {
		return nil, errs.NewInternalServerError("failed to build in-memory upload", err)
	}
	if err := writer.Close(); err != nil {
		return nil, errs.NewInternalServerError("failed to build in-memory upload", err)
	}

	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(int64(len(data)) + 1024)
	if err != nil {
		return nil, errs.NewInternalServerError("failed to build in-memory upload", err)
	}

	files := form.File["file"]
	if len(files) == 0 {
		return nil, errs.NewInternalServerError("failed to build in-memory upload", nil)
	}
	return files[0], nil
}
