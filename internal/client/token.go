package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"terraform-provider-authzed/internal/models"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// TokenWithETag represents a token resource with its ETag
type TokenWithETag struct {
	Token *models.TokenRequest
	ETag  string
}

// GetID returns the token's ID
func (t *TokenWithETag) GetID() string {
	return t.Token.ID
}

// GetETag returns the ETag value
func (t *TokenWithETag) GetETag() string {
	return t.ETag
}

// SetETag sets the ETag value
func (t *TokenWithETag) SetETag(etag string) {
	t.ETag = etag
}

// GetResource returns the underlying token
func (t *TokenWithETag) GetResource() interface{} {
	return t.Token
}

// CreateToken creates a new token
func (c *CloudClient) CreateToken(ctx context.Context, token *models.TokenRequest) (*TokenWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens", token.PermissionsSystemID, token.ServiceAccountID)

	// Create the request body
	reqBody := struct {
		Name        string `json:"name"`
		Description string `json:"description,omitempty"`
	}{
		Name:        token.Name,
		Description: token.Description,
	}

	// Use retry logic with exponential backoff
	retryConfig := DefaultRetryConfig()

	// Define the create operation
	createOperation := func() (*ResponseWithETag, error) {
		req, err := c.NewRequest(http.MethodPost, path, reqBody)
		if err != nil {
			return nil, err
		}

		respWithETag, err := c.Do(req)
		if err != nil {
			return nil, err
		}

		return respWithETag, nil
	}

	// For CREATE operations, we don't need to get fresh ETags since there's no existing resource
	// We just retry the same operation
	getLatestETag := func() (string, error) {
		return "", nil // Not used for CREATE operations
	}

	createWithETag := func(etag string) (*ResponseWithETag, error) {
		return createOperation() // Retry the same CREATE operation
	}

	// Execute with retry logic
	respWithETag, err := retryConfig.RetryWithExponentialBackoffLegacy(
		ctx,
		createOperation,
		getLatestETag,
		createWithETag,
		"token create",
	)
	if err != nil {
		return nil, err
	}

	defer func() {
		// ignore the error
		_ = respWithETag.Response.Body.Close()
	}()

	if respWithETag.Response.StatusCode != http.StatusCreated {
		return nil, NewAPIError(respWithETag)
	}

	// Read the response body once
	body, err := io.ReadAll(respWithETag.Response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// The API response has a different structure during creation
	var createResp struct {
		ID                  string `json:"id"`
		Name                string `json:"name"`
		Description         string `json:"description"`
		PermissionsSystemID string `json:"permissionsSystemId"`
		ServiceAccountID    string `json:"serviceAccountId"`
		CreatedAt           string `json:"createdAt"`
		Creator             string `json:"creator"`
		UpdatedAt           string `json:"updatedAt"`
		Updater             string `json:"updater"`
		Hash                string `json:"hash"`
		Secret              string `json:"secret"`
		ConfigETag          string `json:"ConfigETag"`
	}

	if err := json.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// Convert to our model structure
	tokenModel := &models.TokenRequest{
		ID:                  createResp.ID,
		Name:                createResp.Name,
		Description:         createResp.Description,
		PermissionsSystemID: createResp.PermissionsSystemID,
		ServiceAccountID:    createResp.ServiceAccountID,
		CreatedAt:           createResp.CreatedAt,
		Creator:             createResp.Creator,
		Hash:                createResp.Hash,
		Secret:              createResp.Secret,
	}

	return &TokenWithETag{
		Token: tokenModel,
		ETag:  respWithETag.ETag,
	}, nil
}

func (c *CloudClient) GetToken(permissionsSystemID, serviceAccountID, tokenID string) (*TokenWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens/%s", permissionsSystemID, serviceAccountID, tokenID)

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

	// Read the response body once
	body, err := io.ReadAll(respWithETag.Response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Log the raw response for debugging
	tflog.Debug(context.Background(), fmt.Sprintf("Raw API response: %s", string(body)))

	var token models.TokenRequest
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &TokenWithETag{
		Token: &token,
		ETag:  respWithETag.ETag,
	}, nil
}

func (c *CloudClient) ListTokens(permissionsSystemID, serviceAccountID string) ([]models.TokenRequest, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens", permissionsSystemID, serviceAccountID)
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
		Items []models.TokenRequest `json:"items"`
	}
	if err := json.NewDecoder(respWithETag.Response.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Items, nil
}

// UpdateToken updates an existing token using PUT
func (c *CloudClient) UpdateToken(ctx context.Context, token *models.TokenRequest, etag string) (*TokenWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens/%s", token.PermissionsSystemID, token.ServiceAccountID, token.ID)

	resourceWrapper := &TokenWithETag{
		Token: token,
		ETag:  etag,
	}

	updatedResource, err := c.UpdateResource(ctx, resourceWrapper, path, token)
	if err != nil {
		return nil, err
	}

	return updatedResource.(*TokenWithETag), nil
}

// DeleteToken deletes a token by ID for a service account
func (c *CloudClient) DeleteToken(permissionsSystemID, serviceAccountID, tokenID string) error {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens/%s", permissionsSystemID, serviceAccountID, tokenID)
	return c.DeleteResource(path)
}
