package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type CloudClient struct {
	Host       string
	Token      string
	APIVersion string
	HTTPClient *http.Client
}

type CloudClientConfig struct {
	Host       string
	Token      string
	APIVersion string
	Timeout    time.Duration
}

// NewCloudClient creates a new Cloud API client
func NewCloudClient(cfg *CloudClientConfig) *CloudClient {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = DefaultTimeout
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

	// Extract the ETag from standard header
	etag := resp.Header.Get("ETag")

	return &ResponseWithETag{
		Response: resp,
		ETag:     etag,
	}, nil
}

// UpdateResource updates any resource that implements the Resource interface
func (c *CloudClient) UpdateResource(resource Resource, endpoint string, body any) (Resource, error) {
	// Check if the ETag is empty
	if resource.GetETag() == "" {
		// We only update the ETag without updating local attributes. ETags represent specific
		// versions of resources (like a hash) and this is purely for optimistic concurrency control.
		// If the remote resource differs from local state, the update will fail with 412 and retry with latest version.
		req, err := c.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, err
		}

		respWithETag, err := c.Do(req)
		if err != nil {
			return nil, err
		}

		if respWithETag.Response.StatusCode != http.StatusOK {
			_ = respWithETag.Response.Body.Close()
			return nil, fmt.Errorf("failed to fetch current resource for ETag: %s", NewAPIError(respWithETag))
		}

		// Extract the ETag from the GET response
		etag := respWithETag.ETag

		// Set the ETag on the resource
		resource.SetETag(etag)

		_ = respWithETag.Response.Body.Close()
	}

	// Proceed with the update using the ETag (original or newly fetched)
	req, err := c.NewRequest(http.MethodPut, endpoint, body, WithETag(resource.GetETag()))
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

	if respWithETag.Response.StatusCode != http.StatusNoContent {
		return NewAPIError(respWithETag)
	}

	return nil
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
func (c *CloudClient) CreateResourceWithFactory(endpoint string, body any, dest any, factory ResourceFactory) (Resource, error) {
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

	// Use the factory to create a Resource from the decoded object
	// The factory will handle extracting ConfigETag from the response body if needed
	return factory(dest, respWithETag.ETag), nil
}
