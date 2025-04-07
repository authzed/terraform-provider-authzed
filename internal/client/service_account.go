package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"terraform-provider-platform-api/internal/models"
)

func (c *CloudClient) ListServiceAccounts(permissionSystemID string) ([]models.ServiceAccount, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts", permissionSystemID)

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

	// Read the entire body
	bodyBytes, err := io.ReadAll(resp.Body)
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

func (c *CloudClient) GetServiceAccount(permissionSystemID, serviceAccountID string) (*models.ServiceAccount, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s", permissionSystemID, serviceAccountID)

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

	var serviceAccount models.ServiceAccount
	if err := json.NewDecoder(resp.Body).Decode(&serviceAccount); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &serviceAccount, nil
}

func (c *CloudClient) CreateServiceAccount(serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts", serviceAccount.PermissionSystemID)

	// Debug logging
	fmt.Printf("CreateServiceAccount called with:\n")
	fmt.Printf("  - Host: %s\n", c.Host)
	fmt.Printf("  - Path: %s\n", path)
	fmt.Printf("  - Full URL will be: %s%s\n", c.Host, path)
	fmt.Printf("  - PermissionSystemID: %s\n", serviceAccount.PermissionSystemID)
	fmt.Printf("  - Name: %s\n", serviceAccount.Name)
	fmt.Printf("  - Description: %s\n", serviceAccount.Description)

	// Marshal service account to JSON for logging
	jsonBytes, _ := json.Marshal(serviceAccount)
	fmt.Printf("Request body: %s\n", string(jsonBytes))

	req, err := c.NewRequest(http.MethodPost, path, serviceAccount)
	if err != nil {
		return nil, err
	}

	// Log the fully constructed URL
	fmt.Printf("Final request URL: %s\n", req.URL.String())
	fmt.Printf("Request headers: %+v\n", req.Header)

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, NewAPIError(resp)
	}

	var createdServiceAccount models.ServiceAccount
	if err := json.NewDecoder(resp.Body).Decode(&createdServiceAccount); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdServiceAccount, nil
}

func (c *CloudClient) DeleteServiceAccount(permissionSystemID, serviceAccountID string) error {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s", permissionSystemID, serviceAccountID)

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
