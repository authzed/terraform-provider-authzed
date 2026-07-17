package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff/v4"

	"terraform-provider-authzed/internal/models"
)

// NewPollBackOff returns the exponential backoff used to poll long-running
// Materialize operations: 250ms initial interval capped at 5s, bounded by
// the caller's context rather than an elapsed-time limit.
func NewPollBackOff() *backoff.ExponentialBackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = 250 * time.Millisecond
	b.MaxInterval = 5 * time.Second
	b.MaxElapsedTime = 0
	return b
}

// doMaterializeRequest sends a request to a Materialize endpoint with the
// internal API version. Idempotent methods retry on 429/5xx with backoff; a
// POST is only retried on 429, which guarantees the request was not
// processed — retrying an ambiguous 5xx could create a duplicate deployment
// that Terraform would never track. It must never retry 404 (used as a
// signal by existence polling) or 409/412 (the Materialize API has no ETag
// semantics).
func (c *CloudClient) doMaterializeRequest(ctx context.Context, method, path string, body any) (*ResponseWithETag, error) {
	retryable := func(status int) bool {
		if status == http.StatusTooManyRequests {
			return true
		}
		return method != http.MethodPost && status >= 500
	}

	var resp *ResponseWithETag
	operation := func() error {
		req, err := c.NewRequest(method, path, body, WithAPIVersion(MaterializeAPIVersion))
		if err != nil {
			return backoff.Permanent(err)
		}
		req = req.WithContext(ctx)

		result, err := c.Do(req)
		if err != nil {
			// A network error is ambiguous for a non-idempotent request;
			// fail fast rather than risk a duplicate.
			return backoff.Permanent(err)
		}
		if retryable(result.Response.StatusCode) {
			apiErr := NewAPIError(result)
			_ = result.Response.Body.Close()
			return apiErr
		}
		resp = result
		return nil
	}

	b := backoff.WithMaxRetries(NewPollBackOff(), uint64(DefaultMaxRetries))
	if err := backoff.Retry(operation, backoff.WithContext(b, ctx)); err != nil {
		return nil, err
	}
	return resp, nil
}

// CreateMaterializeDeployment creates a Materialize deployment for a
// permission system deployment
func (c *CloudClient) CreateMaterializeDeployment(ctx context.Context, createReq *models.CreateMaterializeDeploymentRequest) (*models.CreateMaterializeDeploymentResponse, error) {
	resp, err := c.doMaterializeRequest(ctx, http.MethodPost, "/materialize", createReq)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Response.Body.Close()
	}()

	if resp.Response.StatusCode != http.StatusCreated {
		return nil, NewAPIError(resp)
	}

	var created models.CreateMaterializeDeploymentResponse
	if err := json.NewDecoder(resp.Response.Body).Decode(&created); err != nil {
		return nil, fmt.Errorf("failed to decode create materialize deployment response: %w", err)
	}
	return &created, nil
}

// GetMaterializeDeployment fetches a Materialize deployment by its mc-... ID
func (c *CloudClient) GetMaterializeDeployment(ctx context.Context, id string) (*models.MaterializeDeployment, error) {
	resp, err := c.doMaterializeRequest(ctx, http.MethodGet, fmt.Sprintf("/materialize/%s", id), nil)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Response.Body.Close()
	}()

	if resp.Response.StatusCode != http.StatusOK {
		return nil, NewAPIError(resp)
	}

	var getResp models.GetMaterializeDeploymentResponse
	if err := json.NewDecoder(resp.Response.Body).Decode(&getResp); err != nil {
		return nil, fmt.Errorf("failed to decode materialize deployment response: %w", err)
	}
	return &getResp.Deployment, nil
}

// UpdateMaterializeDeployment updates a Materialize deployment's
// configuration. The response carries no body, so callers should re-fetch
// the deployment to observe the applied configuration.
func (c *CloudClient) UpdateMaterializeDeployment(ctx context.Context, id string, updateReq *models.UpdateMaterializeDeploymentRequest) error {
	resp, err := c.doMaterializeRequest(ctx, http.MethodPatch, fmt.Sprintf("/materialize/%s", id), updateReq)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Response.Body.Close()
	}()

	if resp.Response.StatusCode != http.StatusOK && resp.Response.StatusCode != http.StatusNoContent {
		return NewAPIError(resp)
	}
	return nil
}

// DeleteMaterializeDeployment deletes a Materialize deployment. The API
// answers 202 Accepted and deprovisions asynchronously; deletion also
// removes the deployment's authorization relationships, after which the
// API's permission check rejects reads of the deleted deployment with 403
// instead of 404. A 403 on the DELETE itself is still a real permission
// error — only after the DELETE was accepted does 403 mean "gone", which is
// why this does not reuse the shared DeleteResource polling (its poll loop
// treats 403 as a hard error).
func (c *CloudClient) DeleteMaterializeDeployment(id string) error {
	endpoint := fmt.Sprintf("/materialize/%s", id)

	// DELETE is idempotent, so it rides the same 429/5xx retry funnel as the
	// other verbs.
	respWithETag, err := c.doMaterializeRequest(context.Background(), http.MethodDelete, endpoint, nil)
	if err != nil {
		return err
	}
	defer func() {
		_ = respWithETag.Response.Body.Close()
	}()

	switch status := respWithETag.Response.StatusCode; status {
	case http.StatusOK, http.StatusNoContent:
		return nil
	case http.StatusNotFound, http.StatusGone:
		// Idempotent delete success.
		return nil
	case http.StatusAccepted:
		return c.waitForMaterializeDeletion(endpoint)
	default:
		return NewAPIError(respWithETag)
	}
}

// waitForMaterializeDeletion polls the deployment endpoint after an accepted
// DELETE until it reads as gone. 404/410 mean the record is deleted; 403
// means the deployment's authorization relationships were removed as part of
// deletion (the caller was authorized moments ago, so a permission failure
// here cannot mean anything else). Everything retryable (2xx while
// deprovisioning, 409/412/429, 5xx, network errors) keeps polling within
// the client's DeleteTimeout.
func (c *CloudClient) waitForMaterializeDeletion(endpoint string) error {
	ctx, cancel := context.WithTimeout(context.Background(), c.DeleteTimeout)
	defer cancel()

	operation := func() error {
		probeCtx, probeCancel := context.WithTimeout(ctx, 15*time.Second)
		defer probeCancel()

		req, err := c.NewRequest(http.MethodGet, endpoint, nil, WithAPIVersion(MaterializeAPIVersion))
		if err != nil {
			return backoff.Permanent(fmt.Errorf("failed to create request while polling for deletion: %w", err))
		}
		req = req.WithContext(probeCtx)

		respWithETag, err := c.Do(req)
		if err != nil {
			// Network errors are retryable within the delete timeout.
			return err
		}
		defer func() {
			_ = respWithETag.Response.Body.Close()
		}()

		status := respWithETag.Response.StatusCode
		switch {
		// Gone: deleted record (404/410) or deleted authz relationships (403).
		case status == http.StatusNotFound || status == http.StatusGone || status == http.StatusForbidden:
			return nil
		// Retryable: still present (2xx) or 409/412/429/5xx.
		case (status >= 200 && status < 300) || status == http.StatusConflict || status == http.StatusPreconditionFailed || status == http.StatusTooManyRequests || status >= 500:
			return fmt.Errorf("materialize deployment still present (status %d)", status)
		default:
			return backoff.Permanent(fmt.Errorf("unexpected status code %d while polling for materialize deployment deletion", status))
		}
	}

	if err := backoff.Retry(operation, backoff.WithContext(NewPollBackOff(), ctx)); err != nil {
		return fmt.Errorf("waiting for materialize deployment deletion at %s: %w", endpoint, err)
	}
	return nil
}
