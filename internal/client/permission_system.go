package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"terraform-provider-cloud-api/internal/models"
)

// ListPermissionSystems retrieves all permission systems
func (c *CloudClient) ListPermissionSystems() ([]models.PermissionSystem, error) {
	path := "/ps"
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

	// Read response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the raw response for debugging
	fmt.Printf("DEBUG: Raw API response for listing permission systems: %s\n", string(bodyBytes))

	// Create a new reader from the bytes for JSON decoding
	bodyReader := bytes.NewReader(bodyBytes)

	// Try direct array decoding first
	var permissionSystems []models.PermissionSystem
	if err := json.NewDecoder(bodyReader).Decode(&permissionSystems); err != nil {
		// If direct array decoding fails, try with the wrapper that has "items" field
		bodyReader.Seek(0, io.SeekStart) // Reset reader to beginning
		var listResp struct {
			Items []models.PermissionSystem `json:"items"`
		}
		if err := json.NewDecoder(bodyReader).Decode(&listResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return listResp.Items, nil
	}

	return permissionSystems, nil
}

// GetPermissionSystem retrieves a permission system by ID
func (c *CloudClient) GetPermissionSystem(permissionSystemID string) (*models.PermissionSystem, error) {
	path := fmt.Sprintf("/ps/%s", permissionSystemID)
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

	// Read response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the raw response for debugging
	fmt.Printf("DEBUG: Raw API response for permission system %s: %s\n", permissionSystemID, string(bodyBytes))

	// Create a new reader from the bytes for JSON decoding
	bodyReader := bytes.NewReader(bodyBytes)

	// Try direct decoding first
	var permissionSystem models.PermissionSystem
	if err := json.NewDecoder(bodyReader).Decode(&permissionSystem); err != nil {
		// If direct decoding fails, try with the wrapper
		bodyReader.Seek(0, io.SeekStart) // Reset reader to beginning
		var getResp struct {
			PermissionSystem models.PermissionSystem `json:"permissionSystem"`
		}
		if err := json.NewDecoder(bodyReader).Decode(&getResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &getResp.PermissionSystem, nil
	}

	return &permissionSystem, nil
}
