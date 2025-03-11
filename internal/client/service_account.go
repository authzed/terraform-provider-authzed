package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"terraform-provider-platform-api/internal/models"
)

func (c *PlatformClient) ListServiceAccounts(permissionSystemID string) ([]models.ServiceAccount, error) {
	req, err := c.NewRequest(http.MethodGet, fmt.Sprintf("/access/service-accounts?permissionSystemID=%s", permissionSystemID), nil)
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
		Items []models.ServiceAccount `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Items, nil
}

func (c *PlatformClient) GetServiceAccount(permissionSystemID, serviceAccountID string) (*models.ServiceAccount, error) {
	req, err := c.NewRequest(http.MethodGet, fmt.Sprintf("/access/service-accounts/%s?permissionSystemID=%s", serviceAccountID, permissionSystemID), nil)
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

func (c *PlatformClient) CreateServiceAccount(serviceAccount *models.ServiceAccount) (*models.ServiceAccount, error) {
	req, err := c.NewRequest(http.MethodPost, "/access/service-accounts", serviceAccount)
	if err != nil {
		return nil, err
	}

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

func (c *PlatformClient) DeleteServiceAccount(permissionSystemID, serviceAccountID string) error {
	req, err := c.NewRequest(http.MethodDelete, fmt.Sprintf("/access/service-accounts/%s?permissionSystemID=%s", serviceAccountID, permissionSystemID), nil)
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
