package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"terraform-provider-authzed/internal/models"
)

func (c *CloudClient) CreateToken(token *models.Token) (*models.Token, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens", token.PermissionsSystemID, token.ServiceAccountID)
	req, err := c.NewRequest(http.MethodPost, path, token)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// ignore the error
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		return nil, NewAPIError(resp)
	}

	var createdToken models.Token
	if err := json.NewDecoder(resp.Body).Decode(&createdToken); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdToken, nil
}

func (c *CloudClient) GetToken(permissionsSystemID, serviceAccountID, tokenID string) (*models.Token, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens/%s", permissionsSystemID, serviceAccountID, tokenID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// ignore the error
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, NewAPIError(resp)
	}

	var token models.Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &token, nil
}

func (c *CloudClient) ListTokens(permissionsSystemID, serviceAccountID string) ([]models.Token, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens", permissionsSystemID, serviceAccountID)
	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		// ignore the error
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, NewAPIError(resp)
	}

	var listResp struct {
		Items []models.Token `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Items, nil
}

// DeleteToken deletes a token by ID for a service account
func (c *CloudClient) DeleteToken(permissionsSystemID, serviceAccountID, tokenID string) error {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens/%s", permissionsSystemID, serviceAccountID, tokenID)
	req, err := c.NewRequest(http.MethodDelete, path, nil)
	if err != nil {
		return err
	}

	resp, err := c.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		// ignore the error
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusNoContent {
		return NewAPIError(resp)
	}

	return nil
}
