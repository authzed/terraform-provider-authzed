package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"terraform-provider-platform-api/internal/models"
)

// ListRoles retrieves all roles for a ps
func (c *PlatformClient) ListRoles(permissionSystemID string) ([]models.Role, error) {
	req, err := c.NewRequest(http.MethodGet, fmt.Sprintf("/access/roles?permissionSystemID=%s", permissionSystemID), nil)
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
		Items []models.Role `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Items, nil
}

// GetRole retrieves a role by ID
func (c *PlatformClient) GetRole(permissionSystemID, roleID string) (*models.Role, error) {
	req, err := c.NewRequest(http.MethodGet, fmt.Sprintf("/access/roles/%s?permissionSystemID=%s", roleID, permissionSystemID), nil)
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

	var role models.Role
	if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &role, nil
}

func (c *PlatformClient) CreateRole(role *models.Role) (*models.Role, error) {
	req, err := c.NewRequest(http.MethodPost, "/access/roles", role)
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

	var createdRole models.Role
	if err := json.NewDecoder(resp.Body).Decode(&createdRole); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdRole, nil
}

func (c *PlatformClient) DeleteRole(permissionSystemID, roleID string) error {
	req, err := c.NewRequest(http.MethodDelete, fmt.Sprintf("/access/roles/%s?permissionSystemID=%s", roleID, permissionSystemID), nil)
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
