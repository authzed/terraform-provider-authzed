package client

import (
	"bytes"
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
func (c *CloudClient) CreatePolicy(policy *models.Policy) (*PolicyWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/policies", policy.PermissionsSystemID)

	var createdPolicy models.Policy
	resource, err := c.CreateResourceWithFactory(path, policy, &createdPolicy, NewPolicyResource)
	if err != nil {
		return nil, err
	}

	return resource.(*PolicyWithETag), nil
}

// UpdatePolicy updates an existing policy
func (c *CloudClient) UpdatePolicy(policy *models.Policy, etag string) (*PolicyWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/policies/%s", policy.PermissionsSystemID, policy.ID)

	// Create a direct PUT request without using UpdateResource
	req, err := c.NewRequest(http.MethodPut, path, policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Only set If-Match header if we have a non-empty ETag
	if etag != "" {
		req.Header.Set("If-Match", etag)
	}

	respWithETag, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
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

	if respWithETag.Response.StatusCode != http.StatusOK {
		// If it's a 404 error, attempt to recreate the resource
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
				return nil, fmt.Errorf("failed to recreate policy: %s", NewAPIError(createResp))
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

		return nil, fmt.Errorf("failed to update policy: %s", NewAPIError(respWithETag))
	}

	// Decode the updated policy from the response
	var updatedPolicy models.Policy
	if err := json.NewDecoder(io.NopCloser(bytes.NewBuffer(respBody))).Decode(&updatedPolicy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Create and return the wrapped policy with ETag
	return &PolicyWithETag{
		Policy: &updatedPolicy,
		ETag:   respWithETag.ETag,
	}, nil
}

// DeletePolicy deletes a policy by its ID
func (c *CloudClient) DeletePolicy(permissionsSystemID, policyID string) error {
	path := fmt.Sprintf("/ps/%s/access/policies/%s", permissionsSystemID, policyID)
	return c.DeleteResource(path)
}
