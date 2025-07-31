package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"terraform-provider-authzed/internal/models"
)

// PermissionsSystemWithETag represents a permissions system resource with its ETag
type PermissionsSystemWithETag struct {
	PermissionsSystem *models.PermissionsSystem
	ETag              string
}

// GetID returns the permissions system's ID
func (ps *PermissionsSystemWithETag) GetID() string {
	return ps.PermissionsSystem.ID
}

// GetETag returns the ETag value
func (ps *PermissionsSystemWithETag) GetETag() string {
	return ps.ETag
}

// SetETag sets the ETag value
func (ps *PermissionsSystemWithETag) SetETag(etag string) {
	ps.ETag = etag
}

// GetResource returns the underlying permissions system
func (ps *PermissionsSystemWithETag) GetResource() any {
	return ps.PermissionsSystem
}

// ListPermissionsSystems retrieves all permission systems
func (c *CloudClient) ListPermissionsSystems() ([]models.PermissionsSystem, error) {
	path := "/ps"
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	respWithETag, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// Ignore the error
		_ = respWithETag.Response.Body.Close()
	}()

	if respWithETag.Response.StatusCode != http.StatusOK {
		return nil, NewAPIError(respWithETag)
	}

	// Read response body for debugging
	bodyBytes, err := io.ReadAll(respWithETag.Response.Body)
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
func (c *CloudClient) GetPermissionsSystem(permissionsSystemID string) (*PermissionsSystemWithETag, error) {
	path := fmt.Sprintf("/ps/%s", permissionsSystemID)

	var permissionsSystem models.PermissionsSystem
	resource, err := c.GetResourceWithFactory(path, &permissionsSystem, NewPermissionsSystemResource)
	if err != nil {
		return nil, err
	}

	// Log the raw response for debugging
	fmt.Printf("DEBUG: Successfully retrieved permission system %s\n", permissionsSystemID)

	return resource.(*PermissionsSystemWithETag), nil
}
