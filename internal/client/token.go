package client

import (
	"encoding/json"
	"fmt"
	"net/http"

	"terraform-provider-authzed/internal/models"
)

// TokenWithETag represents a token resource with its ETag
type TokenWithETag struct {
	Token *models.Token
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

func (c *CloudClient) CreateToken(token *models.Token) (*TokenWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens", token.PermissionsSystemID, token.ServiceAccountID)

	var createdToken models.Token
	resource, err := c.CreateResourceWithFactory(path, token, &createdToken, NewTokenResource)
	if err != nil {
		return nil, err
	}

	return resource.(*TokenWithETag), nil
}

func (c *CloudClient) GetToken(permissionsSystemID, serviceAccountID, tokenID string) (*TokenWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens/%s", permissionsSystemID, serviceAccountID, tokenID)

	var token models.Token
	resource, err := c.GetResourceWithFactory(path, &token, NewTokenResource)
	if err != nil {
		return nil, err
	}

	return resource.(*TokenWithETag), nil
}

func (c *CloudClient) ListTokens(permissionsSystemID, serviceAccountID string) ([]models.Token, error) {
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
		Items []models.Token `json:"items"`
	}
	if err := json.NewDecoder(respWithETag.Response.Body).Decode(&listResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return listResp.Items, nil
}

// UpdateToken updates an existing token using PUT
func (c *CloudClient) UpdateToken(token *models.Token, etag string) (*TokenWithETag, error) {
	path := fmt.Sprintf("/ps/%s/access/service-accounts/%s/tokens/%s", token.PermissionsSystemID, token.ServiceAccountID, token.ID)

	resourceWrapper := &TokenWithETag{
		Token: token,
		ETag:  etag,
	}

	updatedResource, err := c.UpdateResource(resourceWrapper, path, token)
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
