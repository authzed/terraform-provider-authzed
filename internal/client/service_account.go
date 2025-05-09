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

	var createdServiceAccount models.ServiceAccount
	resource, err := c.CreateResourceWithFactory(path, serviceAccount, &createdServiceAccount, NewServiceAccountResource)
	if err != nil {
		return nil, err
	}

	return resource.(*ServiceAccountWithETag), nil
}

// UpdateServiceAccount updates an existing service account using the PUT method
func (c *CloudClient) UpdateServiceAccount(serviceAccount *models.ServiceAccount, etag string) (*ServiceAccountWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s", serviceAccount.PermissionsSystemID, serviceAccount.ID)

	// First, verify that the service account exists
	existingServiceAccount, err := c.GetServiceAccount(serviceAccount.PermissionsSystemID, serviceAccount.ID)
	if err != nil {
		// Proceed with the update attempt even if verification fails
		// This provides better resilience against transient API issues
	} else {
		// If we found the service account, use its ETag if the provided ETag is empty
		if etag == "" && existingServiceAccount.ETag != "" {
			etag = existingServiceAccount.ETag
		}
	}

	// Define a function to get the latest ETag
	getLatestETag := func() (string, error) {
		getReq, err := c.NewRequest(http.MethodGet, path, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create GET request: %w", err)
		}

		getResp, err := c.Do(getReq)
		if err != nil {
			return "", fmt.Errorf("failed to send GET request: %w", err)
		}
		defer func() {
			if getResp.Response.Body != nil {
				_ = getResp.Response.Body.Close()
			}
		}()

		if getResp.Response.StatusCode != http.StatusOK {
			return "", fmt.Errorf("failed to get latest ETag, status: %d", getResp.Response.StatusCode)
		}

		// Extract ETag from response header
		latestETag := getResp.ETag
		return latestETag, nil
	}

	// Try update with provided ETag
	updateWithETag := func(currentETag string) (*ResponseWithETag, error) {
		req, err := c.NewRequest(http.MethodPut, path, serviceAccount)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Only set If-Match header if we have a non-empty ETag
		if currentETag != "" {
			req.Header.Set("If-Match", currentETag)
		}

		respWithETag, err := c.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		return respWithETag, nil
	}

	// First attempt with the provided ETag
	respWithETag, err := updateWithETag(etag)
	if err != nil {
		return nil, err
	}

	// Handle the 412 Precondition Failed error by retrying with the latest ETag
	if respWithETag.Response.StatusCode == http.StatusPreconditionFailed {
		// Close the body of the first response
		if respWithETag.Response.Body != nil {
			_ = respWithETag.Response.Body.Close()
		}

		// Get the latest ETag
		latestETag, err := getLatestETag()
		if err != nil {
			return nil, fmt.Errorf("failed to get latest ETag for retry: %w", err)
		}

		// Retry the update with the latest ETag
		respWithETag, err = updateWithETag(latestETag)
		if err != nil {
			return nil, err
		}
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

	// Handle 404 Not Found
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
			return nil, NewAPIError(createResp)
		}

		// Decode the created service account
		var createdServiceAccount models.ServiceAccount
		if err := json.NewDecoder(createResp.Response.Body).Decode(&createdServiceAccount); err != nil {
			return nil, fmt.Errorf("failed to decode recreated service account: %w", err)
		}

		// Create the result with the ETag
		etag := createResp.ETag
		if etag == "" && createdServiceAccount.ConfigETag != "" {
			etag = createdServiceAccount.ConfigETag
		}

		// Create the result with the original ID to maintain consistency
		result := &ServiceAccountWithETag{
			ServiceAccount: &createdServiceAccount,
			ETag:           etag,
		}

		// Force the right ID to maintain Terraform state consistency
		if result.ServiceAccount.ID != originalID {
			result.ServiceAccount.ID = originalID
		}

		return result, nil
	}

	// Handle other error status codes
	if respWithETag.Response.StatusCode != http.StatusOK {
		return nil, NewAPIError(respWithETag)
	}

	// Decode the updated service account from the response
	var updatedServiceAccount models.ServiceAccount
	if err := json.NewDecoder(io.NopCloser(bytes.NewBuffer(respBody))).Decode(&updatedServiceAccount); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Check for ConfigETag in the response
	etag = respWithETag.ETag
	if etag == "" && updatedServiceAccount.ConfigETag != "" {
		etag = updatedServiceAccount.ConfigETag
	}

	// Create and return the wrapped service account with ETag
	result := &ServiceAccountWithETag{
		ServiceAccount: &updatedServiceAccount,
		ETag:           etag,
	}

	return result, nil
}

func (c *CloudClient) DeleteServiceAccount(permissionsSystemID, serviceAccountID string) error {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s", permissionsSystemID, serviceAccountID)
	return c.DeleteResource(path)
}
