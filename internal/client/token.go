package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"terraform-provider-platform-api/internal/models"
)

func (c *PlatformClient) CreateToken(token *models.Token) (*models.Token, error) {
	req, err := c.NewRequest(http.MethodPost, fmt.Sprintf("/access/service-accounts/%s/tokens", token.ServiceAccountID), token)
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

	var createdToken models.Token
	if err := json.NewDecoder(resp.Body).Decode(&createdToken); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdToken, nil
}

func (c *PlatformClient) GetToken(permissionSystemID, serviceAccountID, tokenID string) (*models.Token, error) {
	req, err := c.NewRequest(http.MethodGet, fmt.Sprintf("/access/service-accounts/%s/tokens/%s?permissionSystemID=%s", serviceAccountID, tokenID, permissionSystemID), nil)
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

	var token models.Token
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &token, nil
}

func (c *PlatformClient) ListTokens(permissionSystemID, serviceAccountID string) ([]models.Token, error) {
	req, err := c.NewRequest(http.MethodGet, fmt.Sprintf("/access/service-accounts/%s/tokens?permissionSystemID=%s", serviceAccountID, permissionSystemID), nil)
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

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Try to decode as a direct array first
	var tokens []models.Token
	if err := json.Unmarshal(bodyBytes, &tokens); err != nil {
		// If direct decode fails, try with the wrapped items format
		var listResp struct {
			Items []models.Token `json:"items"`
		}
		if err := json.Unmarshal(bodyBytes, &listResp); err != nil {
			return nil, fmt.Errorf("failed to decode response: %w", err)
		}
		return listResp.Items, nil
	}

	return tokens, nil
}

// DeleteToken deletes a token by ID for a service account
func (c *PlatformClient) DeleteToken(permissionSystemID, serviceAccountID, tokenID string) error {
	req, err := c.NewRequest(http.MethodDelete, fmt.Sprintf("/access/service-accounts/%s/tokens/%s?permissionSystemID=%s", serviceAccountID, tokenID, permissionSystemID), nil)
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
