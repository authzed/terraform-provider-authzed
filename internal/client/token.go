package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/terraform-plugin-log/tflog"

	"terraform-provider-authzed/internal/models"
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
func (t *TokenWithETag) GetResource() any {
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

	// Setup idempotent recovery
	recovery := &IdempotentRecoveryConfig{
		ResourceType: "token",
		LookupByName: func(name string) (Resource, error) {
			tokens, err := c.ListTokens(token.PermissionsSystemID, token.ServiceAccountID)
			if err != nil {
				return nil, err
			}
			for _, t := range tokens {
				if t.Name == name {
					// Get the full resource with ETag
					return c.GetToken(ctx, token.PermissionsSystemID, token.ServiceAccountID, t.ID)
				}
			}
			return nil, nil
		},
	}

	// Use retry logic with exponential backoff
	retryConfig := DefaultRetryConfig()
	// Broaden retry conditions for CREATE: include 5xx and 404 (SA not visible yet) as retryable
	previousShouldRetry := retryConfig.ShouldRetry
	retryConfig.ShouldRetry = func(statusCode int) bool {
		if statusCode >= 500 {
			return true
		}
		// Retry 404s for token creation, service account may not be globally visible yet
		if statusCode == http.StatusNotFound {
			return true
		}
		return previousShouldRetry(statusCode)
	}

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

		// Treat 409/429/5xx as retryable by converting to APIError
		if respWithETag.Response != nil {
			code := respWithETag.Response.StatusCode
			if code == http.StatusConflict || code == http.StatusTooManyRequests || code >= 500 {
				return nil, NewAPIError(respWithETag)
			}
		}

		return respWithETag, nil
	}

	// For CREATE operations, we don't need to get fresh ETags since there's no existing resource
	getLatestETag := func() (string, error) {
		return "", nil // Not used for CREATE operations
	}

	createWithETag := func(etag string) (*ResponseWithETag, error) { //nolint:unparam
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
		// Attempt idempotent recovery for ambiguous outcomes
		if isAmbiguousError(err) {
			if recovered, recErr := recovery.RecoverFromAmbiguousCreate(ctx, token.Name, err); recErr == nil {
				return recovered.(*TokenWithETag), nil
			}
		}
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
	}

	if err := json.Unmarshal(body, &createResp); err != nil {
		return nil, fmt.Errorf("failed to decode token creation response: %w (response body: %s)", err, string(body))
	}

	// Validate required fields in the response
	if createResp.ID == "" {
		return nil, fmt.Errorf("token creation response missing required field 'id' (response body: %s)", string(body))
	}
	if createResp.Secret == "" {
		return nil, fmt.Errorf("token creation response missing required field 'secret' (response body: %s)", string(body))
	}
	if createResp.PermissionsSystemID == "" {
		return nil, fmt.Errorf("token creation response missing required field 'permissionsSystemId' (response body: %s)", string(body))
	}
	if createResp.ServiceAccountID == "" {
		return nil, fmt.Errorf("token creation response missing required field 'serviceAccountId' (response body: %s)", string(body))
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

	// Create initial resource
	resource := &TokenWithETag{
		Token: tokenModel,
		ETag:  respWithETag.ETag,
	}

	return resource, nil
}

func (c *CloudClient) GetToken(ctx context.Context, permissionsSystemID, serviceAccountID, tokenID string) (*TokenWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens/%s", permissionsSystemID, serviceAccountID, tokenID)

	req, err := c.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	// Apply context to the request
	req = req.WithContext(ctx)

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
		return nil, fmt.Errorf("failed to decode token GET response: %w (response body: %s)", err, string(body))
	}

	// Validate required fields in the GET response
	if token.ID == "" {
		return nil, fmt.Errorf("token GET response missing required field 'id' (response body: %s)", string(body))
	}
	if token.PermissionsSystemID == "" {
		return nil, fmt.Errorf("token GET response missing required field 'permissionsSystemID' (response body: %s)", string(body))
	}
	if token.ServiceAccountID == "" {
		return nil, fmt.Errorf("token GET response missing required field 'serviceAccountID' (response body: %s)", string(body))
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

	// Read the response body for better error reporting
	body, err := io.ReadAll(respWithETag.Response.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var listResp struct {
		Items []models.TokenRequest `json:"items"`
	}
	if err := json.Unmarshal(body, &listResp); err != nil {
		return nil, fmt.Errorf("failed to decode token list response: %w (response body: %s)", err, string(body))
	}

	// Validate the response structure
	if listResp.Items == nil {
		return []models.TokenRequest{}, nil // Return empty slice instead of nil
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
