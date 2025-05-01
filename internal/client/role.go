package client

import (
	"encoding/json"
	"fmt"
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

	resourceWrapper := &RoleWithETag{
		Role: role,
		ETag: etag,
	}

	updatedResource, err := c.UpdateResource(resourceWrapper, path, role)
	if err != nil {
		return nil, err
	}

	return updatedResource.(*RoleWithETag), nil
}

// DeleteRole deletes a role by its ID
func (c *CloudClient) DeleteRole(permissionsSystemID, roleID string) error {
	path := fmt.Sprintf("/ps/%s/access/roles/%s", permissionsSystemID, roleID)
	return c.DeleteResource(path)
}
