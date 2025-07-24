package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CloudClient is the HTTP client for the AuthZed Cloud API
type CloudClient struct {
	Host          string
	Token         string
	APIVersion    string
	HTTPClient    *http.Client
	DeleteTimeout time.Duration
}

// CloudClientConfig represents the config for the Cloud API client
type CloudClientConfig struct {
	Host          string
	Token         string
	APIVersion    string
	Timeout       time.Duration
	DeleteTimeout time.Duration
}

// NewCloudClient creates a new Cloud API client
func NewCloudClient(cfg *CloudClientConfig) *CloudClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
	}

	deleteTimeout := cfg.DeleteTimeout
	if deleteTimeout == 0 {
		deleteTimeout = DefaultDeleteTimeout
	}

	apiVersion := cfg.APIVersion
	if apiVersion == "" {
		apiVersion = DefaultAPIVersion
	}

	return &CloudClient{
		Host:       cfg.Host,
		Token:      cfg.Token,
		APIVersion: apiVersion,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
		DeleteTimeout: deleteTimeout,
	}
}

// NewRequest creates a new HTTP request with the necessary headers
func (c *CloudClient) NewRequest(method, path string, body any, options ...RequestOption) (*http.Request, error) {
	// Fix URL construction to handle trailing slashes properly
	host := c.Host
	// Remove trailing slash from host if path starts with slash
	if len(host) > 0 && host[len(host)-1] == '/' && len(path) > 0 && path[0] == '/' {
		host = host[:len(host)-1]
	}
	url := fmt.Sprintf("%s%s", host, path)

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

// Do sends an HTTP request and returns an HTTP response with the ETag if present
func (c *CloudClient) Do(req *http.Request) (*ResponseWithETag, error) {
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

// UpdateResource updates any resource that implements the Resource interface
func (c *CloudClient) UpdateResource(ctx context.Context, resource Resource, endpoint string, body any) (Resource, error) {
	// Define a function to get the latest ETag
	getLatestETag := func() (string, error) {
		req, err := c.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create GET request: %w", err)
		}

		resp, err := c.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to send GET request: %w", err)
		}
		defer func() {
			if resp.Response.Body != nil {
				_ = resp.Response.Body.Close()
			}
		}()

		if resp.Response.StatusCode != http.StatusOK {
			return "", NewAPIError(resp)
		}

		return resp.ETag, nil
	}

	// Define a function to update with a specific ETag
	updateWithETag := func(currentETag string) (*ResponseWithETag, error) {
		req, err := c.NewRequest(http.MethodPut, endpoint, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set the If-Match header for optimistic concurrency control
		req.Header.Set("If-Match", currentETag)

		resp, err := c.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		if resp.Response.StatusCode != http.StatusOK {
			defer func() {
				if resp.Response.Body != nil {
					_ = resp.Response.Body.Close()
				}
			}()
			return nil, NewAPIError(resp)
		}

		return resp, nil
	}

	// Use retry logic with exponential backoff
	retryConfig := DefaultRetryConfig()
	respWithETag, err := retryConfig.RetryWithExponentialBackoffLegacy(
		ctx,
		func() (*ResponseWithETag, error) {
			return updateWithETag(resource.GetETag())
		},
		getLatestETag,
		updateWithETag,
		"resource update",
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		if respWithETag.Response.Body != nil {
			_ = respWithETag.Response.Body.Close()
		}
	}()

	// Update the resource's ETag
	resource.SetETag(respWithETag.ETag)

	return resource, nil
}

// GetResource retrieves a resource by its endpoint and unmarshals it into the provided destination
func (c *CloudClient) GetResource(endpoint string, dest any) (Resource, error) {
	req, err := c.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	respWithETag, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// ignore the error
		_ = respWithETag.Response.Body.Close()
	}()

	if respWithETag.Response.StatusCode != http.StatusOK {
		return nil, NewAPIError(respWithETag)
	}

	if err := json.NewDecoder(respWithETag.Response.Body).Decode(dest); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// The caller needs to convert the dest into a Resource implementation
	// and set the ETag on it before returning
	return nil, nil
}

// CreateResource creates a new resource
func (c *CloudClient) CreateResource(endpoint string, body any, dest any) (Resource, error) {
	req, err := c.NewRequest(http.MethodPost, endpoint, body)
	if err != nil {
		return nil, err
	}

	respWithETag, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// ignore the error
		_ = respWithETag.Response.Body.Close()
	}()

	if respWithETag.Response.StatusCode != http.StatusCreated {
		return nil, NewAPIError(respWithETag)
	}

	if err := json.NewDecoder(respWithETag.Response.Body).Decode(dest); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// The caller needs to convert the dest into a Resource implementation
	// and set the ETag on it before returning
	return nil, nil
}

// DeleteResource deletes a resource
func (c *CloudClient) DeleteResource(endpoint string) error {
	req, err := c.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}

	respWithETag, err := c.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		// ignore the error
		_ = respWithETag.Response.Body.Close()
	}()

	// Handle synchronous delete (204 No Content)
	if respWithETag.Response.StatusCode == http.StatusNoContent {
		return nil
	}

	// Handle asynchronous delete (202 Accepted)
	if respWithETag.Response.StatusCode == http.StatusAccepted {
		return c.waitForDeletion(endpoint)
	}

	// All other status codes are errors
	return NewAPIError(respWithETag)
}

// waitForDeletion polls the resource endpoint until it returns 404 (deleted)
func (c *CloudClient) waitForDeletion(endpoint string) error {
	timeout := time.After(c.DeleteTimeout)
	ticker := time.NewTicker(DefaultDeletePollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for resource deletion at %s (waited %v)", endpoint, c.DeleteTimeout)
		case <-ticker.C:
			// Check if resource still exists
			req, err := c.NewRequest(http.MethodGet, endpoint, nil)
			if err != nil {
				return fmt.Errorf("failed to create request while polling for deletion: %w", err)
			}

			respWithETag, err := c.Do(req)
			if err != nil {
				return fmt.Errorf("failed to poll for deletion: %w", err)
			}

			// Close response body immediately
			_ = respWithETag.Response.Body.Close()

			// 404 means resource is deleted - success!
			if respWithETag.Response.StatusCode == http.StatusNotFound {
				return nil
			}

			// 200 means resource still exists - continue polling
			if respWithETag.Response.StatusCode == http.StatusOK {
				continue
			}

			// Any other status code is unexpected
			return fmt.Errorf("unexpected status code %d while polling for deletion", respWithETag.Response.StatusCode)
		}
	}
}

// ResourceFactory is a function that creates a Resource from the decoded response
type ResourceFactory func(decoded any, etag string) Resource

// GetResourceWithFactory combines GetResource with a factory to create a proper Resource
func (c *CloudClient) GetResourceWithFactory(endpoint string, dest any, factory ResourceFactory) (Resource, error) {
	req, err := c.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	respWithETag, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// ignore the error
		_ = respWithETag.Response.Body.Close()
	}()

	if respWithETag.Response.StatusCode != http.StatusOK {
		return nil, NewAPIError(respWithETag)
	}

	if err := json.NewDecoder(respWithETag.Response.Body).Decode(dest); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Use the factory to create a Resource from the decoded object
	return factory(dest, respWithETag.ETag), nil
}

// CreateResourceWithFactory combines CreateResource with a factory to create a proper Resource
func (c *CloudClient) CreateResourceWithFactory(ctx context.Context, endpoint string, body any, dest any, factory ResourceFactory) (Resource, error) {
	// Use retry logic with exponential backoff
	retryConfig := DefaultRetryConfig()

	// Define the create operation
	createOperation := func() (*ResponseWithETag, error) {
		req, err := c.NewRequest(http.MethodPost, endpoint, body)
		if err != nil {
			return nil, err
		}

		respWithETag, err := c.Do(req)
		if err != nil {
			return nil, err
		}

		return respWithETag, nil
	}

	// For CREATE operations, we don't need to get fresh ETags since there's no existing resource
	// We just retry the same operation
	getLatestETag := func() (string, error) {
		return "", nil // Not used for CREATE operations
	}

	createWithETag := func(etag string) (*ResponseWithETag, error) {
		return createOperation() // Retry the same CREATE operation
	}

	// Execute with retry logic
	respWithETag, err := retryConfig.RetryWithExponentialBackoffLegacy(
		ctx,
		createOperation,
		getLatestETag,
		createWithETag,
		"resource create",
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		// ignore the error
		_ = respWithETag.Response.Body.Close()
	}()

	if respWithETag.Response.StatusCode != http.StatusCreated {
		return nil, NewAPIError(respWithETag)
	}

	if err := json.NewDecoder(respWithETag.Response.Body).Decode(dest); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Use the factory to create a Resource from the decoded object
	return factory(dest, respWithETag.ETag), nil
}
