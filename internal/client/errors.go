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

type HTTPResponseWrapper struct {
	*http.Response
}

func (r *HTTPResponseWrapper) GetResponse() *http.Response {
	return r.Response
}

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
		// Check if this is a configuration conflict and provide helpful context
		if e.StatusCode == 409 && containsFGAMConfigConflict(e.Message) {
			return fmt.Sprintf("API error (status %d): %s\n\nThis error occurs when the Fine-Grained Access Management (FGAM) configuration for the permission system has been modified by another process. The Terraform provider will automatically retry this operation.", e.StatusCode, e.Message)
		}
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (status %d)", e.StatusCode)
}

// containsFGAMConfigConflict checks if the error message indicates a configuration conflict
func containsFGAMConfigConflict(message string) bool {
	lowerMessage := strings.ToLower(message)

	return strings.Contains(lowerMessage, "restricted api access configuration") &&
		strings.Contains(lowerMessage, "has changed")
}

// errorDetails pulls the per-problem messages out of an error body. The
// top-level message is often a generic summary ("bad request"), while these
// say which field is wrong and why.
func errorDetails(jsonErr map[string]any, summary string) string {
	raw, ok := jsonErr["errors"].([]any)
	if !ok {
		return ""
	}
	details := make([]string, 0, len(raw))
	for _, item := range raw {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		message, ok := entry["message"].(string)
		// Skip a detail that only repeats the summary, so it is not said twice.
		if !ok || message == "" || message == summary {
			continue
		}
		details = append(details, message)
	}
	return strings.Join(details, "; ")
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
		if details := errorDetails(jsonErr, errMsg); details != "" {
			if errMsg == "" {
				errMsg = details
			} else {
				errMsg += ": " + details
			}
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
