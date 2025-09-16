package client

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strings"
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

	// Configure transport based on env flags
	transport := &http.Transport{}

	// Force HTTP/1.1 if requested (optional debugging flag)
	if os.Getenv("AUTHZED_HTTP1") == "1" {
		transport.ForceAttemptHTTP2 = false
		transport.TLSNextProto = map[string]func(string, *tls.Conn) http.RoundTripper{}
	}

	// Default: disable compression to preserve ETag headers reliably.
	// Explicitly enable compression only when AUTHZED_DISABLE_GZIP=0.
	if os.Getenv("AUTHZED_DISABLE_GZIP") == "0" {
		transport.DisableCompression = false
	} else {
		transport.DisableCompression = true
	}

	return &CloudClient{
		Host:       cfg.Host,
		Token:      cfg.Token,
		APIVersion: apiVersion,
		HTTPClient: &http.Client{
			Timeout:   timeout,
			Transport: transport,
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
	// Only set Content-Type when a body is present
	if bodyReader != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	// Set User-Agent (configurable for debugging)
	userAgent := os.Getenv("AUTHZED_USER_AGENT")
	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	// Send Accept-Encoding: identity by default. Allow compression only when AUTHZED_DISABLE_GZIP=0
	if os.Getenv("AUTHZED_DISABLE_GZIP") != "0" {
		req.Header.Set("Accept-Encoding", "identity")
	}

	// Apply any provided options
	for _, option := range options {
		option(req)
	}

	return req, nil
}

// ResponseWithETag wraps an HTTP response and its ETag
type ResponseWithETag struct {
	Response *http.Response
	ETag     string
}

// RequestOption allows setting optional parameters for requests
type RequestOption func(*http.Request)

// Do sends an HTTP request and returns an HTTP response with the ETag if present
func (c *CloudClient) Do(req *http.Request) (*ResponseWithETag, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}

	// Extract the ETag per OpenAPI specification
	etag := resp.Header.Get("ETag")

	// Debug logging (optional)
	if os.Getenv("AUTHZED_DEBUG_HEADERS") == "1" {
		fmt.Printf("DEBUG: Protocol: %s\n", resp.Proto)
		fmt.Printf("DEBUG: Status: %s\n", resp.Status)
		fmt.Printf("DEBUG: All response headers: %+v\n", resp.Header)
		fmt.Printf("DEBUG: Raw ETag value: '%s'\n", etag)
	}

	return &ResponseWithETag{
		Response: resp,
		ETag:     etag,
	}, nil
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

	status := respWithETag.Response.StatusCode
	// 200 OK or 204 No Content: synchronous delete success
	if status == http.StatusOK || status == http.StatusNoContent {
		if os.Getenv("AUTHZED_DELETE_CONFIRM_ON_204") == "1" {
			confirmCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if req2, err := c.NewRequest(http.MethodGet, endpoint, nil); err == nil {
				req2 = req2.WithContext(confirmCtx)
				if resp2, err2 := c.Do(req2); err2 == nil {
					_ = resp2.Response.Body.Close()
				}
			}
		}
		return nil
	}
	// 404 Not Found: idempotent delete success
	if status == http.StatusNotFound {
		return nil
	}
	// 202 Accepted: async delete, poll for completion
	if status == http.StatusAccepted {
		return c.waitForDeletion(endpoint)
	}

	// Other statuses are errors
	return NewAPIError(respWithETag)
}

// waitForDeletion polls the resource endpoint until it returns 404/410 (deleted)
func (c *CloudClient) waitForDeletion(endpoint string) error {
	// Overall timeout
	ctx, cancel := context.WithTimeout(context.Background(), c.DeleteTimeout)
	defer cancel()

	attempt := 0
	base := 250 * time.Millisecond
	capDelay := 5 * time.Second
	start := time.Now()

	for {
		if ctx.Err() != nil {
			return fmt.Errorf("timeout waiting for resource deletion at %s (waited %v)", endpoint, c.DeleteTimeout)
		}

		// Short per-probe timeout
		probeCtx, probeCancel := context.WithTimeout(ctx, 15*time.Second)
		req, err := c.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			probeCancel()
			return fmt.Errorf("failed to create request while polling for deletion: %w", err)
		}
		req = req.WithContext(probeCtx)

		respWithETag, err := c.Do(req)
		probeCancel()
		attempt++

		var status int
		if err != nil {
			// Treat network errors as retryable within timeout
			status = 0
		} else {
			status = respWithETag.Response.StatusCode
			_ = respWithETag.Response.Body.Close()
		}

		if os.Getenv("AUTHZED_DEBUG_DELETE") == "1" {
			fmt.Printf("DEBUG delete poll: path=%s attempt=%d elapsed=%s status=%d err=%v\n", endpoint, attempt, time.Since(start).String(), status, err)
		}

		// Success terminal states
		if status == http.StatusNotFound || status == http.StatusGone {
			return nil
		}

		// Retryable states: 2xx (still present), 409, 412, 429, 5xx, or network error (status==0)
		if (status >= 200 && status < 300) || status == http.StatusConflict || status == http.StatusPreconditionFailed || status == http.StatusTooManyRequests || status >= 500 || status == 0 {
			// backoff with jitter
			delay := backoffDelay(base, capDelay, attempt)
			time.Sleep(delay)
			continue
		}

		// Non-retryable 4xx (other than 404/410)
		return fmt.Errorf("unexpected status code %d while polling for deletion", status)
	}
}

func backoffDelay(base, cap time.Duration, attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	exp := time.Duration(float64(base) * math.Pow(2, float64(attempt-1)))
	if exp > cap {
		exp = cap
	}
	jitter := time.Duration(rand.Int63n(int64(exp) / 2))
	return exp + jitter
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

	// Header-only ETag stabilization: retry up to 3 times if header missing
	etag := respWithETag.ETag
	if etag == "" {
		for i := 0; i < 3 && etag == ""; i++ {
			if retryReq, rerr := c.NewRequest(http.MethodGet, endpoint, nil); rerr == nil {
				if retryResp, derr := c.Do(retryReq); derr == nil {
					etag = retryResp.ETag
					if retryResp.Response.Body != nil {
						_ = retryResp.Response.Body.Close()
					}
				}
			}
			if etag == "" {
				time.Sleep(200 * time.Millisecond)
			}
		}
	}

	return factory(dest, etag), nil
}

// GetResourceWithFactoryWithContext is the context-aware version of GetResourceWithFactory
func (c *CloudClient) GetResourceWithFactoryWithContext(ctx context.Context, endpoint string, dest any, factory ResourceFactory) (Resource, error) {
	req, err := c.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	// Apply context to the request
	req = req.WithContext(ctx)

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
		return nil, err
	}

	// Header-only ETag stabilization: retry up to 3 times if header missing
	etag := respWithETag.ETag
	if etag == "" {
		for i := 0; i < 3 && etag == ""; i++ {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			default:
			}

			if retryReq, rerr := c.NewRequest(http.MethodGet, endpoint, nil); rerr == nil {
				retryReq = retryReq.WithContext(ctx)
				if retryResp, derr := c.Do(retryReq); derr == nil {
					etag = retryResp.ETag
					if retryResp.Response.Body != nil {
						_ = retryResp.Response.Body.Close()
					}
				}
			}
		}
	}

	return factory(dest, etag), nil
}

// CreateResourceWithFactory combines CreateResource with a factory to create a proper Resource
func (c *CloudClient) CreateResourceWithFactory(ctx context.Context, endpoint string, body any, dest any, factory ResourceFactory) (Resource, error) {
	return c.CreateResourceWithFactoryAndRecovery(ctx, endpoint, body, dest, factory, nil)
}

// CreateResourceWithFactoryAndRecovery creates a resource with optional idempotent recovery
func (c *CloudClient) CreateResourceWithFactoryAndRecovery(ctx context.Context, endpoint string, body any, dest any, factory ResourceFactory, recovery *IdempotentRecoveryConfig) (Resource, error) {
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

		// Retryable statuses for CREATE: 409/429/5xx
		if respWithETag.Response != nil {
			code := respWithETag.Response.StatusCode
			if code == http.StatusConflict || code == http.StatusTooManyRequests || code >= 500 {
				// Return as error to trigger retry wrapper
				return nil, NewAPIError(respWithETag)
			}
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
		// Attempt idempotent recovery for ambiguous outcomes
		if recovery != nil && isAmbiguousError(err) {
			if bodyMap, ok := body.(map[string]interface{}); ok {
				if name, exists := bodyMap["name"].(string); exists {
					if recovered, recErr := recovery.RecoverFromAmbiguousCreate(ctx, name, err); recErr == nil {
						return recovered, nil
					}
				}
			}
		}
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

	// Create resource from response - no stabilization needed for simple resources
	resource := factory(dest, respWithETag.ETag)

	// Skip stabilization if ETag is already present (resource is immediately ready)
	if resource.GetETag() != "" {
		return resource, nil
	}

	// Only run stabilization for resources that need it (missing ETag)
	stabilizationConfig := DefaultStabilizationConfig()
	getResourceForStabilization := func() (Resource, error) {
		return c.GetResourceWithFactory(endpoint, dest, factory)
	}

	stabilizedResource, stabilizationErr := stabilizationConfig.WaitForResourceReadiness(
		ctx,
		getResourceForStabilization,
		"created resource",
	)
	if stabilizationErr != nil {
		// Return original resource if stabilization fails but creation succeeded
		return resource, nil
	}

	return stabilizedResource, nil
}

// isAmbiguousError checks if an error represents an ambiguous outcome
func isAmbiguousError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "504") ||
		strings.Contains(errStr, "502") ||
		strings.Contains(errStr, "503")
}
