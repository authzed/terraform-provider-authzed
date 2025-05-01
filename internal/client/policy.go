package client

import (
	"encoding/json"
	"fmt"
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

// UpdatePolicy updates an existing policy using the PUT method
func (c *CloudClient) UpdatePolicy(policy *models.Policy, etag string) (*PolicyWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/policies/%s", policy.PermissionsSystemID, policy.ID)

	resourceWrapper := &PolicyWithETag{
		Policy: policy,
		ETag:   etag,
	}

	updatedResource, err := c.UpdateResource(resourceWrapper, path, policy)
	if err != nil {
		return nil, err
	}

	return updatedResource.(*PolicyWithETag), nil
}

// DeletePolicy deletes a policy by its ID
func (c *CloudClient) DeletePolicy(permissionsSystemID, policyID string) error {
	path := fmt.Sprintf("/ps/%s/access/policies/%s", permissionsSystemID, policyID)
	return c.DeleteResource(path)
}
