package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"terraform-provider-authzed/internal/models"
)

// ServiceAccountWithETag represents a service account resource with its ETag
type ServiceAccountWithETag struct {
	ServiceAccount *models.ServiceAccount
	ETag           string
}

// GetID returns the service account's ID
func (sa *ServiceAccountWithETag) GetID() string {
	return sa.ServiceAccount.ID
}

// GetETag returns the ETag value
func (sa *ServiceAccountWithETag) GetETag() string {
	return sa.ETag
}

// SetETag sets the ETag value
func (sa *ServiceAccountWithETag) SetETag(etag string) {
	sa.ETag = etag
}

// GetResource returns the underlying service account
func (sa *ServiceAccountWithETag) GetResource() interface{} {
	return sa.ServiceAccount
}

func (c *CloudClient) ListServiceAccounts(permissionsSystemID string) ([]models.ServiceAccount, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts", permissionsSystemID)

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

	// Read the entire body
	bodyBytes, err := io.ReadAll(respWithETag.Response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to decode as a direct array first
	var serviceAccounts []models.ServiceAccount
	if err := json.Unmarshal(bodyBytes, &serviceAccounts); err != nil {
		// If direct decode fails, try with the wrapped items format
		var listResp struct {
			Items []models.ServiceAccount `json:"items"`
		}
		if err := json.Unmarshal(bodyBytes, &listResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return listResp.Items, nil
	}

	return serviceAccounts, nil
}

// GetServiceAccount retrieves a service account by ID
func (c *CloudClient) GetServiceAccount(permissionsSystemID, serviceAccountID string) (*ServiceAccountWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s", permissionsSystemID, serviceAccountID)

	var serviceAccount models.ServiceAccount
	resource, err := c.GetResourceWithFactory(path, &serviceAccount, NewServiceAccountResource)
	if err != nil {
		return nil, err
	}

	// Type assertion to convert Resource back to the concrete type
	return resource.(*ServiceAccountWithETag), nil
}

func (c *CloudClient) CreateServiceAccount(serviceAccount *models.ServiceAccount) (*ServiceAccountWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts", serviceAccount.PermissionsSystemID)

	// Debug logging
	fmt.Printf("CreateServiceAccount called with:\n")
	fmt.Printf("  - Host: %s\n", c.Host)
	fmt.Printf("  - Path: %s\n", path)
	fmt.Printf("  - Full URL will be: %s%s\n", c.Host, path)
	fmt.Printf("  - PermissionsSystemID: %s\n", serviceAccount.PermissionsSystemID)
	fmt.Printf("  - Name: %s\n", serviceAccount.Name)
	fmt.Printf("  - Description: %s\n", serviceAccount.Description)

	// Marshal service account to JSON for logging
	jsonBytes, _ := json.Marshal(serviceAccount)
	fmt.Printf("Request body: %s\n", string(jsonBytes))

	var createdServiceAccount models.ServiceAccount
	resource, err := c.CreateResourceWithFactory(path, serviceAccount, &createdServiceAccount, NewServiceAccountResource)
	if err != nil {
		return nil, err
	}

	// Log the request URL
	fmt.Printf("Request was successful\n")

	return resource.(*ServiceAccountWithETag), nil
}

// UpdateServiceAccount updates an existing service account using the PUT method
func (c *CloudClient) UpdateServiceAccount(serviceAccount *models.ServiceAccount, etag string) (*ServiceAccountWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s", serviceAccount.PermissionsSystemID, serviceAccount.ID)

	// Create a direct PUT request without using UpdateResource
	req, err := c.NewRequest(http.MethodPut, path, serviceAccount)
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
			// Recreate the service account using POST to the base endpoint
			createPath := fmt.Sprintf("/ps/%s/access/service-accounts", serviceAccount.PermissionsSystemID)
			originalID := serviceAccount.ID

			createReq, err := c.NewRequest(http.MethodPost, createPath, serviceAccount)
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
				return nil, fmt.Errorf("failed to recreate service account: %s", NewAPIError(createResp))
			}

			// Decode the created service account
			var createdServiceAccount models.ServiceAccount
			if err := json.NewDecoder(createResp.Response.Body).Decode(&createdServiceAccount); err != nil {
				return nil, fmt.Errorf("failed to decode recreated service account: %w", err)
			}

			// Create the result with the original ID to maintain consistency
			result := &ServiceAccountWithETag{
				ServiceAccount: &createdServiceAccount,
				ETag:           createResp.ETag,
			}

			// Force the right ID to maintain Terraform state consistency
			if result.ServiceAccount.ID != originalID {
				result.ServiceAccount.ID = originalID
			}

			return result, nil
		}

		return nil, fmt.Errorf("failed to update service account: %s", NewAPIError(respWithETag))
	}

	// Decode the updated service account from the response
	var updatedServiceAccount models.ServiceAccount
	if err := json.NewDecoder(io.NopCloser(bytes.NewBuffer(respBody))).Decode(&updatedServiceAccount); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Create and return the wrapped service account with ETag
	return &ServiceAccountWithETag{
		ServiceAccount: &updatedServiceAccount,
		ETag:           respWithETag.ETag,
	}, nil
}

func (c *CloudClient) DeleteServiceAccount(permissionsSystemID, serviceAccountID string) error {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s", permissionsSystemID, serviceAccountID)
	return c.DeleteResource(path)
}
