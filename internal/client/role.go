package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"terraform-provider-cloud-api/internal/models"
)

// ListRoles retrieves all roles for a ps
func (c *PlatformClient) ListRoles(permissionSystemID string) ([]models.Role, error) {
	path := fmt.Sprintf("/ps/%s/access/roles", permissionSystemID)

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
		Items []models.Role `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Items, nil
}

// GetRole retrieves a role by ID
func (c *PlatformClient) GetRole(permissionSystemID, roleID string) (*models.Role, error) {
	path := fmt.Sprintf("/ps/%s/access/roles/%s", permissionSystemID, roleID)

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

	var role models.Role
	if err := json.NewDecoder(resp.Body).Decode(&role); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &role, nil
}

func (c *PlatformClient) CreateRole(role *models.Role) (*models.Role, error) {
	path := fmt.Sprintf("/ps/%s/access/roles", role.PermissionSystemID)

	req, err := c.NewRequest(http.MethodPost, path, role)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Read the response body to check for specific error messages
		body, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == http.StatusInternalServerError {
			// Check if the error message indicates a duplicate name
			if strings.Contains(string(body), "duplicate") || strings.Contains(string(body), "already exists") {
				return nil, fmt.Errorf("role with name '%s' already exists in permission system '%s'", role.Name, role.PermissionSystemID)
			}
		}
		// If it's not a duplicate name error, return the original API error
		return nil, NewAPIError(resp)
	}

	var createdRole models.Role
	if err := json.NewDecoder(resp.Body).Decode(&createdRole); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdRole, nil
}

func (c *PlatformClient) DeleteRole(permissionSystemID, roleID string) error {
	path := fmt.Sprintf("/ps/%s/access/roles/%s", permissionSystemID, roleID)

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
