package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"terraform-provider-platform-api/internal/models"
)

// ListPermissionSystems retrieves all permiss
func (c *PlatformClient) ListPermissionSystems() ([]models.PermissionSystem, error) {
	req, err := c.NewRequest(http.MethodGet, "/ps", nil)
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
		Items []models.PermissionSystem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Items, nil
}

// GetPermissionSystem retrieves a ps by ID
func (c *PlatformClient) GetPermissionSystem(id string) (*models.PermissionSystem, error) {
	req, err := c.NewRequest(http.MethodGet, fmt.Sprintf("/ps/%s", id), nil)
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

	var getResp struct {
		PermissionSystem models.PermissionSystem `json:"permissionSystem"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&getResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &getResp.PermissionSystem, nil
}
