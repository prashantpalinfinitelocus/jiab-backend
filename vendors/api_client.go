package vendors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Infinite-Locus-Product/coca_cola_backend/utils"
	"github.com/moul/http2curl"
	log "github.com/sirupsen/logrus"
)

// APIClient represents a generic HTTP client for external APIs
type APIClient struct {
	BaseURL string
	Client  *http.Client
	Headers map[string]string
	Timeout time.Duration
}

// APIResponse represents a generic API response
type APIResponse struct {
	StatusCode int
	Body       []byte
	Headers    http.Header
}

// Initialize creates and configures a new API client
func InitializeClient(baseURL string, headers map[string]string) *APIClient {
	return &APIClient{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
		Headers: headers,
		Timeout: 30 * time.Second,
	}
}

// SetTimeout sets the request timeout for the client
func (c *APIClient) SetTimeout(timeout time.Duration) {
	c.Timeout = timeout
	c.Client.Timeout = timeout
}

// SetHeader sets a custom header for all requests
func (c *APIClient) SetHeader(key, value string) {
	c.Headers[key] = value
}

// buildURL constructs the full URL with the endpoint
func (c *APIClient) buildURL(endpoint string) string {
	endpoint = strings.TrimPrefix(endpoint, "/")
	return fmt.Sprintf("%s/%s", c.BaseURL, endpoint)
}

// makeRequest performs the actual HTTP request
func (c *APIClient) makeRequest(method, endpoint string, body interface{}, queryParams map[string]string, skipHeaderCanonicalization bool) (*APIResponse, error) {
	fullURL := c.buildURL(endpoint)

	// Add query parameters
	if len(queryParams) > 0 {
		u, err := url.Parse(fullURL)
		if err != nil {
			return nil, fmt.Errorf("invalid URL: %w", err)
		}

		q := u.Query()
		for key, value := range queryParams {
			q.Add(key, value)
		}
		u.RawQuery = q.Encode()
		fullURL = u.String()
	}

	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	for key, value := range c.Headers {
		if skipHeaderCanonicalization {
			req.Header[key] = []string{value}
		} else {
			req.Header.Set(key, value)
		}
	}

	// Generate and log cURL command
	if curlCmd, err := http2curl.GetCurlCommand(req); err == nil {
		log.WithField("curl", utils.SanitizeForLog(curlCmd.String())).Info("Generated cURL command")
	}

	sanitizedHeaders := make(map[string]string, len(c.Headers))
	for k, v := range c.Headers {
		sanitizedHeaders[utils.SanitizeForLog(k)] = utils.SanitizeForLog(v)
	}

	log.WithFields(log.Fields{
		"method":  method,
		"url":     utils.SanitizeForLog(fullURL),
		"headers": sanitizedHeaders,
	}).Debug("Making API request")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	apiResponse := &APIResponse{
		StatusCode: resp.StatusCode,
		Body:       responseBody,
		Headers:    resp.Header,
	}

	log.WithFields(log.Fields{
		"status_code":   resp.StatusCode,
		"response_size": len(responseBody),
	}).Debug("API response received")

	log.WithField("response_body", string(responseBody)).Debug(" ::: Response body")

	return apiResponse, nil
}

// Get performs a GET request
func (c *APIClient) Get(endpoint string, queryParams map[string]string) (*APIResponse, error) {
	return c.makeRequest("GET", endpoint, nil, queryParams, false)
}

// Post performs a POST request
func (c *APIClient) Post(endpoint string, body interface{}, queryParams map[string]string, skipHeaderCanonicalization bool) (*APIResponse, error) {
	return c.makeRequest("POST", endpoint, body, queryParams, false)
}

// Put performs a PUT request
func (c *APIClient) Put(endpoint string, body interface{}, queryParams map[string]string) (*APIResponse, error) {
	return c.makeRequest("PUT", endpoint, body, queryParams, false)
}

// Patch performs a PATCH request
func (c *APIClient) Patch(endpoint string, body interface{}, queryParams map[string]string) (*APIResponse, error) {
	return c.makeRequest("PATCH", endpoint, body, queryParams, false)
}

// Delete performs a DELETE request
func (c *APIClient) Delete(endpoint string, queryParams map[string]string) (*APIResponse, error) {
	return c.makeRequest("DELETE", endpoint, nil, queryParams, false)
}

// ParseJSON parses the response body as JSON into the provided interface
func (r *APIResponse) ParseJSON(v interface{}) error {
	return json.Unmarshal(r.Body, v)
}

// IsSuccess checks if the response status code indicates success (2xx)
func (r *APIResponse) IsSuccess() bool {
	return r.StatusCode >= 200 && r.StatusCode < 300
}

// GetBodyAsString returns the response body as a string
func (r *APIResponse) GetBodyAsString() string {
	return string(r.Body)
}
