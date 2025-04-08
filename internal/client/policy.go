package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"terraform-provider-cloudapi/internal/models"
)

// ListPolicies retrieves all policies for a permission system
func (c *CloudClient) ListPolicies(permissionSystemID string) ([]models.Policy, error) {
	path := fmt.Sprintf("/ps/%s/access/policies", permissionSystemID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewAPIError(resp)
	}

	var listResp struct {
		Items []models.Policy `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Items, nil
}

// GetPolicy retrieves a policy by its ID
func (c *CloudClient) GetPolicy(permissionSystemID, policyID string) (*models.Policy, error) {
	path := fmt.Sprintf("/ps/%s/access/policies/%s", permissionSystemID, policyID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, NewAPIError(resp)
	}

	var policy models.Policy
	if err := json.NewDecoder(resp.Body).Decode(&policy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &policy, nil
}

func (c *CloudClient) CreatePolicy(policy *models.Policy) (*models.Policy, error) {
	path := fmt.Sprintf("/ps/%s/access/policies", policy.PermissionSystemID)
	req, err := c.NewRequest(http.MethodPost, path, policy)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, NewAPIError(resp)
	}

	var createdPolicy models.Policy
	if err := json.NewDecoder(resp.Body).Decode(&createdPolicy); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdPolicy, nil
}

func (c *CloudClient) DeletePolicy(permissionSystemID, policyID string) error {
	path := fmt.Sprintf("/ps/%s/access/policies/%s", permissionSystemID, policyID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return NewAPIError(resp)
	}

	return nil
}
