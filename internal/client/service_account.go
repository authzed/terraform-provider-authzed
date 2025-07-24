package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"terraform-provider-authzed/internal/models"

	"github.com/hashicorp/terraform-plugin-framework/diag"
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

// CreateServiceAccount creates a new service account using the POST method
func (c *CloudClient) CreateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount) (*ServiceAccountWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts", serviceAccount.PermissionsSystemID)

	var createdServiceAccount models.ServiceAccount
	resource, err := c.CreateResourceWithFactory(ctx, path, serviceAccount, &createdServiceAccount, NewServiceAccountResource)
	if err != nil {
		return nil, err
	}

	return resource.(*ServiceAccountWithETag), nil
}

// ServiceAccountUpdateResult contains the result of a service account update and any diagnostics
type ServiceAccountUpdateResult struct {
	ServiceAccount *ServiceAccountWithETag
	Diagnostics    diag.Diagnostics
}

// UpdateServiceAccount updates an existing service account using the PUT method
func (c *CloudClient) UpdateServiceAccount(ctx context.Context, serviceAccount *models.ServiceAccount, etag string) *ServiceAccountUpdateResult {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s", serviceAccount.PermissionsSystemID, serviceAccount.ID)

	// Define a function to update with a specific ETag
	updateWithETag := func(currentETag string) (*ResponseWithETag, error) {
		req, err := c.NewRequest(http.MethodPut, path, serviceAccount)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Set the If-Match header for optimistic concurrency control
		req.Header.Set("If-Match", currentETag)

		resp, err := c.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to send request: %w", err)
		}

		if resp.Response.StatusCode != http.StatusOK {
			defer func() {
				if resp.Response.Body != nil {
					_ = resp.Response.Body.Close()
				}
			}()
			return nil, NewAPIError(resp)
		}

		return resp, nil
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
			return "", NewAPIError(getResp)
		}

		return getResp.ETag, nil
	}

	// Use enhanced retry logic with exponential backoff for FGAM conflicts
	retryConfig := DefaultRetryConfig()
	retryResult := retryConfig.RetryWithExponentialBackoff(
		ctx,
		func() (*ResponseWithETag, error) {
			return updateWithETag(etag)
		},
		getLatestETag,
		updateWithETag,
		"service account update",
	)

	if retryResult.Response == nil {
		return &ServiceAccountUpdateResult{
			ServiceAccount: nil,
			Diagnostics:    retryResult.Diagnostics,
		}
	}

	respWithETag := retryResult.Response
	defer func() {
		if respWithETag.Response.Body != nil {
			_ = respWithETag.Response.Body.Close()
		}
	}()

	var updatedServiceAccount models.ServiceAccount
	if err := json.NewDecoder(respWithETag.Response.Body).Decode(&updatedServiceAccount); err != nil {
		retryResult.Diagnostics.AddError("Decode Error", fmt.Sprintf("Failed to decode response: %v", err))
		return &ServiceAccountUpdateResult{
			ServiceAccount: nil,
			Diagnostics:    retryResult.Diagnostics,
		}
	}

	return &ServiceAccountUpdateResult{
		ServiceAccount: &ServiceAccountWithETag{
			ServiceAccount: &updatedServiceAccount,
			ETag:           respWithETag.ETag,
		},
		Diagnostics: retryResult.Diagnostics,
	}
}

func (c *CloudClient) DeleteServiceAccount(permissionsSystemID, serviceAccountID string) error {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s", permissionsSystemID, serviceAccountID)
	return c.DeleteResource(path)
}
