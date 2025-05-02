package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// PlatformClient is the HTTP client
type PlatformClient struct {
	Host       string
	Token      string
	APIVersion string
	HTTPClient *http.Client
}

// PlatformClientConfig represents the config for the client
type PlatformClientConfig struct {
	Host       string
	Token      string
	APIVersion string
	Timeout    time.Duration
}

// ResponseWithETag wraps an HTTP response and its ETag
type ResponseWithETag struct {
	Response *http.Response
	ETag     string
}

// NewPlatformClient creates a new api client
func NewPlatformClient(cfg *PlatformClientConfig) *PlatformClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	apiVersion := cfg.APIVersion
	if apiVersion == "" {
		apiVersion = DefaultAPIVersion
	}

	return &PlatformClient{
		Host:       cfg.Host,
		Token:      cfg.Token,
		APIVersion: apiVersion,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// NewRequest creates a new HTTP request with the necessary headers
func (c *PlatformClient) NewRequest(method, path string, body any, options ...RequestOption) (*http.Request, error) {
	url := fmt.Sprintf("%s%s", c.Host, path)

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequest(method, url, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.Token))
	req.Header.Set("X-API-Version", c.APIVersion)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Apply any provided options
	for _, option := range options {
		option(req)
	}

	return req, nil
}

// RequestOption allows setting optional parameters for requests
type RequestOption func(*http.Request)

// WithETag adds an If-Match header with the provided ETag
func WithETag(etag string) RequestOption {
	return func(req *http.Request) {
		if etag != "" {
			req.Header.Set("If-Match", etag)
		}
	}
}

// Do sends an HTTP request and returns an HTTP response with the ETag if present
func (c *PlatformClient) Do(req *http.Request) (*ResponseWithETag, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Extract the ETag if it exists
	etag := resp.Header.Get("ETag")

	return &ResponseWithETag{
		Response: resp,
		ETag:     etag,
	}, nil
}
