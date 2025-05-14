package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"terraform-provider-authzed/internal/models"
)

// RoleWithETag represents a role resource with its ETag
type RoleWithETag struct {
	Role *models.Role
	ETag string
}

// GetID returns the role's ID
func (r *RoleWithETag) GetID() string {
	return r.Role.ID
}

// GetETag returns the ETag value
func (r *RoleWithETag) GetETag() string {
	return r.ETag
}

// SetETag sets the ETag value
func (r *RoleWithETag) SetETag(etag string) {
	r.ETag = etag
}

// GetResource returns the underlying role
func (r *RoleWithETag) GetResource() interface{} {
	return r.Role
}

// ListRoles retrieves all roles for a ps
func (c *CloudClient) ListRoles(permissionsSystemID string) ([]models.Role, error) {
	path := fmt.Sprintf("/ps/%s/access/roles", permissionsSystemID)

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
		Items []models.Role `json:"items"`
	}
	if err := json.NewDecoder(respWithETag.Response.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Items, nil
}

// GetRole retrieves a role by ID
func (c *CloudClient) GetRole(permissionsSystemID, roleID string) (*RoleWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/roles/%s", permissionsSystemID, roleID)

	var role models.Role
	resource, err := c.GetResourceWithFactory(path, &role, NewRoleResource)
	if err != nil {
		return nil, err
	}

	return resource.(*RoleWithETag), nil
}

// CreateRole creates a new role
func (c *CloudClient) CreateRole(role *models.Role) (*RoleWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/roles", role.PermissionsSystemID)

	var createdRole models.Role
	resource, err := c.CreateResourceWithFactory(path, role, &createdRole, NewRoleResource)
	if err != nil {
		// Special handling for specific errors
		if apiErr, ok := err.(*APIError); ok && apiErr.StatusCode == http.StatusInternalServerError {
			// Check if the error message indicates a duplicate name
			if strings.Contains(string(apiErr.Body), "duplicate") || strings.Contains(string(apiErr.Body), "already exists") {
				return nil, fmt.Errorf("role with name '%s' already exists in permission system '%s'", role.Name, role.PermissionsSystemID)
			}
		}
		return nil, err
	}

	return resource.(*RoleWithETag), nil
}

// UpdateRole updates an existing role using the PUT method
func (c *CloudClient) UpdateRole(role *models.Role, etag string) (*RoleWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/roles/%s", role.PermissionsSystemID, role.ID)

	// Create a direct PUT request without using UpdateResource
	req, err := c.NewRequest(http.MethodPut, path, role)
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
			// Recreate the role using POST to the base endpoint
			createPath := fmt.Sprintf("/ps/%s/access/roles", role.PermissionsSystemID)
			originalID := role.ID

			createReq, err := c.NewRequest(http.MethodPost, createPath, role)
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

			// Decode the created role
			var createdRole models.Role
			if err := json.NewDecoder(createResp.Response.Body).Decode(&createdRole); err != nil {
				return nil, fmt.Errorf("failed to decode recreated role: %w", err)
			}

			// Create the result with the original ID to maintain consistency
			result := &RoleWithETag{
				Role: &createdRole,
				ETag: createResp.ETag,
			}

			// Force the right ID to maintain Terraform state consistency
			if result.Role.ID != originalID {
				result.Role.ID = originalID
			}

			return result, nil
		}

		return nil, NewAPIError(respWithETag)
	}

	// Decode the updated role from the response
	var updatedRole models.Role
	if err := json.NewDecoder(io.NopCloser(bytes.NewBuffer(respBody))).Decode(&updatedRole); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Create and return the wrapped role with ETag
	return &RoleWithETag{
		Role: &updatedRole,
		ETag: respWithETag.ETag,
	}, nil
}

// DeleteRole deletes a role by its ID
func (c *CloudClient) DeleteRole(permissionsSystemID, roleID string) error {
	path := fmt.Sprintf("/ps/%s/access/roles/%s", permissionsSystemID, roleID)
	return c.DeleteResource(path)
}
