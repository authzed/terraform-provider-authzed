package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"terraform-provider-authzed/internal/models"
)

// PolicyWithETag represents a policy resource with its ETag
type PolicyWithETag struct {
	Policy *models.Policy
	ETag   string
}

// GetID returns the policy's ID
func (p *PolicyWithETag) GetID() string {
	return p.Policy.ID
}

// GetETag returns the ETag value
func (p *PolicyWithETag) GetETag() string {
	return p.ETag
}

// SetETag sets the ETag value
func (p *PolicyWithETag) SetETag(etag string) {
	p.ETag = etag
}

// GetResource returns the underlying policy
func (p *PolicyWithETag) GetResource() interface{} {
	return p.Policy
}

// ListPolicies retrieves all policies for a permission system
func (c *CloudClient) ListPolicies(permissionsSystemID string) ([]models.Policy, error) {
	path := fmt.Sprintf("/ps/%s/access/policies", permissionsSystemID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
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

	var listResp struct {
		Items []models.Policy `json:"items"`
	}
	if err := json.NewDecoder(respWithETag.Response.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Items, nil
}

// GetPolicy retrieves a policy by its ID
func (c *CloudClient) GetPolicy(permissionsSystemID, policyID string) (*PolicyWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/policies/%s", permissionsSystemID, policyID)

	var policy models.Policy
	resource, err := c.GetResourceWithFactory(path, &policy, NewPolicyResource)
	if err != nil {
		return nil, err
	}

	return resource.(*PolicyWithETag), nil
}

// CreatePolicy creates a new policy
func (c *CloudClient) CreatePolicy(ctx context.Context, policy *models.Policy) (*PolicyWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/policies", policy.PermissionsSystemID)

	// Setup idempotent recovery
	recovery := &IdempotentRecoveryConfig{
		ResourceType: "policy",
		LookupByName: func(name string) (Resource, error) {
			policies, err := c.ListPolicies(policy.PermissionsSystemID)
			if err != nil {
				return nil, err
			}
			for _, p := range policies {
				if p.Name == name {
					// Get the full resource with ETag
					return c.GetPolicy(policy.PermissionsSystemID, p.ID)
				}
			}
			return nil, nil
		},
	}

	// Use retry logic with exponential backoff and broadened conditions
	retryConfig := DefaultRetryConfig()
	prevShouldRetry := retryConfig.ShouldRetry
	retryConfig.ShouldRetry = func(statusCode int) bool {
		if statusCode >= 500 { // include 5xx like 502/503/504
			return true
		}
		return prevShouldRetry(statusCode) // 409/412/429
	}

	// Define the create operation
	createOperation := func() (*ResponseWithETag, error) {
		req, err := c.NewRequest(http.MethodPost, path, policy)
		if err != nil {
			return nil, err
		}

		respWithETag, err := c.Do(req)
		if err != nil {
			return nil, err
		}

		// Treat 409/429/5xx as retryable
		if respWithETag.Response != nil {
			code := respWithETag.Response.StatusCode
			if code == http.StatusConflict || code == http.StatusTooManyRequests || code >= 500 {
				return nil, NewAPIError(respWithETag)
			}
		}

		return respWithETag, nil
	}

	// For CREATE operations, we don't need to get fresh ETags
	getLatestETag := func() (string, error) { return "", nil }
	createWithETag := func(etag string) (*ResponseWithETag, error) { return createOperation() }

	// Execute with retry logic
	respWithETag, err := retryConfig.RetryWithExponentialBackoffLegacy(
		ctx,
		createOperation,
		getLatestETag,
		createWithETag,
		"policy create",
	)
	if err != nil {
		// Attempt idempotent recovery for ambiguous outcomes
		if isAmbiguousError(err) {
			if recovered, recErr := recovery.RecoverFromAmbiguousCreate(ctx, policy.Name, err); recErr == nil {
				return recovered.(*PolicyWithETag), nil
			}
		}
		return nil, err
	}
	defer func() {
		if respWithETag.Response.Body != nil {
			_ = respWithETag.Response.Body.Close()
		}
	}()

	if respWithETag.Response.StatusCode != http.StatusCreated {
		return nil, NewAPIError(respWithETag)
	}

	// Decode created policy from POST response
	var createdPolicy models.Policy
	if err := json.NewDecoder(respWithETag.Response.Body).Decode(&createdPolicy); err != nil {
		return nil, fmt.Errorf("failed to decode created policy: %w", err)
	}

	// Create and return the wrapped policy with ETag
	return &PolicyWithETag{
		Policy: &createdPolicy,
		ETag:   respWithETag.ETag,
	}, nil
}

// UpdatePolicy updates an existing policy using the PUT method
func (c *CloudClient) UpdatePolicy(ctx context.Context, policy *models.Policy, etag string) (*PolicyWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/policies/%s", policy.PermissionsSystemID, policy.ID)

	// Define a function to get the latest ETag
	getLatestETag := func() (string, error) {
		getReq, err := c.NewRequest(http.MethodGet, path, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create GET request: %w", err)
		}

		getResp, err := c.Do(getReq)
		if err != nil {
			return "", fmt.Errorf("failed to send GET request: %w", err)
		}
		defer func() {
			if getResp.Response.Body != nil {
				_ = getResp.Response.Body.Close()
			}
		}()

		if getResp.Response.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to get latest ETag, status: %d", getResp.Response.StatusCode)
		}

		// Extract ETag from response header only
		return getResp.ETag, nil
	}

	// Try update with provided ETag
	updateWithETag := func(currentETag string) (*ResponseWithETag, error) {
		req, err := c.NewRequest(http.MethodPut, path, policy)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Only set If-Match header if we have a non-empty ETag
		if currentETag != "" {
			req.Header.Set("If-Match", currentETag)
		}

		respWithETag, err := c.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		return respWithETag, nil
	}

	// Use retry logic with exponential backoff
	retryConfig := DefaultRetryConfig()
	respWithETag, err := retryConfig.RetryWithExponentialBackoffLegacy(
		ctx,
		func() (*ResponseWithETag, error) {
			return updateWithETag(etag)
		},
		getLatestETag,
		updateWithETag,
		"policy update",
	)
	if err != nil {
		return nil, err
	}

	// Keep the response body for potential error reporting
	var respBody []byte
	if respWithETag.Response.Body != nil {
		respBody, _ = io.ReadAll(respWithETag.Response.Body)
		// Replace the body for further use
		respWithETag.Response.Body = io.NopCloser(bytes.NewBuffer(respBody))
	}

	defer func() {
		if respWithETag.Response.Body != nil {
			_ = respWithETag.Response.Body.Close()
		}
	}()

	// Handle 404 Not Found
	if respWithETag.Response.StatusCode == http.StatusNotFound {
		// Recreate the policy using POST to the base endpoint
		createPath := fmt.Sprintf("/ps/%s/access/policies", policy.PermissionsSystemID)
		originalID := policy.ID

		createReq, err := c.NewRequest(http.MethodPost, createPath, policy)
		if err != nil {
			return nil, fmt.Errorf("failed to create request for recreation: %w", err)
		}

		createResp, err := c.Do(createReq)
		if err != nil {
			return nil, fmt.Errorf("failed to send create request for recreation: %w", err)
		}

		defer func() {
			if createResp.Response.Body != nil {
				_ = createResp.Response.Body.Close()
			}
		}()

		if createResp.Response.StatusCode != http.StatusCreated {
			return nil, NewAPIError(createResp)
		}

		// Decode the created policy
		var createdPolicy models.Policy
		if err := json.NewDecoder(createResp.Response.Body).Decode(&createdPolicy); err != nil {
			return nil, fmt.Errorf("failed to decode recreated policy: %w", err)
		}

		// Create the result with the original ID to maintain consistency
		result := &PolicyWithETag{
			Policy: &createdPolicy,
			ETag:   createResp.ETag,
		}

		// Force the right ID to maintain Terraform state consistency
		if result.Policy.ID != originalID {
			result.Policy.ID = originalID
		}

		return result, nil
	}

	// Handle other error status codes
	if respWithETag.Response.StatusCode != http.StatusOK {
		return nil, NewAPIError(respWithETag)
	}

	// Decode the updated policy from the response
	var updatedPolicy models.Policy
	if err := json.NewDecoder(io.NopCloser(bytes.NewBuffer(respBody))).Decode(&updatedPolicy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	etagOut := respWithETag.ETag

	// As a last resort, GET to fetch ETag
	if etagOut == "" {
		fresh, gerr := c.GetPolicy(policy.PermissionsSystemID, policy.ID)
		if gerr == nil && fresh.ETag != "" {
			etagOut = fresh.ETag
		}
	}

	// Create and return the wrapped policy with ETag
	return &PolicyWithETag{
		Policy: &updatedPolicy,
		ETag:   etagOut,
	}, nil
}

// DeletePolicy deletes a policy by its ID
func (c *CloudClient) DeletePolicy(permissionsSystemID, policyID string) error {
	path := fmt.Sprintf("/ps/%s/access/policies/%s", permissionsSystemID, policyID)
	return c.DeleteResource(path)
}
