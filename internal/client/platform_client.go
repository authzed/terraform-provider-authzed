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
func (c *PlatformClient) NewRequest(method, path string, body interface{}) (*http.Request, error) {
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

	return req, nil
}

// Do sends an HTTP request and returns an HTTP response
func (c *PlatformClient) Do(req *http.Request) (*http.Response, error) {
	return c.HTTPClient.Do(req)
}
