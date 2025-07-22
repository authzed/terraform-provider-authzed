package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// HTTPResponder interface for any type that can provide an HTTP response
type HTTPResponder interface {
	GetResponse() *http.Response
}

// Make http.Response implement HTTPResponder
type HTTPResponseWrapper struct {
	*http.Response
}

func (r *HTTPResponseWrapper) GetResponse() *http.Response {
	return r.Response
}

// Make ResponseWithETag implement HTTPResponder
func (r *ResponseWithETag) GetResponse() *http.Response {
	return r.Response
}

type APIError struct {
	StatusCode int
	Message    string
	URL        string
	Method     string
	Body       []byte
}

func (e *APIError) Error() string {
	if e.Message != "" {
		// Check if this is a FGAM configuration conflict and provide helpful context
		if e.StatusCode == 409 && containsFGAMConfigConflict(e.Message) {
			return fmt.Sprintf("API error (status %d): %s\n\nThis error occurs when the Fine-Grained Access Management (FGAM) configuration for the permission system has been modified by another process. The Terraform provider will automatically retry this operation.", e.StatusCode, e.Message)
		}
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (status %d)", e.StatusCode)
}

// containsFGAMConfigConflict checks if the error message indicates a FGAM configuration conflict
func containsFGAMConfigConflict(message string) bool {
	lowerMessage := strings.ToLower(message)

	return strings.Contains(lowerMessage, "restricted api access configuration") &&
		strings.Contains(lowerMessage, "has changed")
}

// NewAPIError creates a new APIError from an HTTPResponder
func NewAPIError(responder HTTPResponder) *APIError {
	resp := responder.GetResponse()
	body, _ := io.ReadAll(resp.Body)

	var errMsg string
	// Try to parse as JSON if possible
	var jsonErr map[string]any
	if err := json.Unmarshal(body, &jsonErr); err == nil {
		if msg, ok := jsonErr["message"].(string); ok {
			errMsg = msg
		} else if msg, ok := jsonErr["error"].(string); ok {
			errMsg = msg
		}
	}

	return &APIError{
		StatusCode: resp.StatusCode,
		Message:    errMsg,
		URL:        resp.Request.URL.String(),
		Method:     resp.Request.Method,
		Body:       body,
	}
}
