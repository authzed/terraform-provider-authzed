package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"terraform-provider-authzed/internal/models"
)

// ListPermissionsSystems retrieves all permission systems
func (c *CloudClient) ListPermissionsSystems() ([]models.PermissionsSystem, error) {
	path := "/ps"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Ignore the error
		_ = resp.Body.Close()
	}()

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
	var permissionsSystems []models.PermissionsSystem
	if err := json.NewDecoder(bodyReader).Decode(&permissionsSystems); err != nil {
		// If direct array decoding fails, try with the wrapper that has "items" field
		_, err = bodyReader.Seek(0, io.SeekStart) // Reset reader to beginning
		if err != nil {
			return nil, err
		}
		var listResp struct {
			Items []models.PermissionsSystem `json:"items"`
		}
		if err := json.NewDecoder(bodyReader).Decode(&listResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return listResp.Items, nil
	}

	return permissionsSystems, nil
}

// GetPermissionsSystem retrieves a permission system by ID
func (c *CloudClient) GetPermissionsSystem(permissionsSystemID string) (*models.PermissionsSystem, error) {
	path := fmt.Sprintf("/ps/%s", permissionsSystemID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Ignore the error
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, NewAPIError(resp)
	}

	// Read response body for debugging
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the raw response for debugging
	fmt.Printf("DEBUG: Raw API response for permission system %s: %s\n", permissionsSystemID, string(bodyBytes))

	// Create a new reader from the bytes for JSON decoding
	bodyReader := bytes.NewReader(bodyBytes)

	// Try direct decoding first
	var permissionsSystem models.PermissionsSystem
	if err := json.NewDecoder(bodyReader).Decode(&permissionsSystem); err != nil {
		// If direct decoding fails, try with the wrapper
		_, err = bodyReader.Seek(0, io.SeekStart) // Reset reader to beginning
		if err != nil {
			return nil, nil
		}
		var getResp struct {
			PermissionsSystem models.PermissionsSystem `json:"permissionsSystem"`
		}
		if err := json.NewDecoder(bodyReader).Decode(&getResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return &getResp.PermissionsSystem, nil
	}

	return &permissionsSystem, nil
}
