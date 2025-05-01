package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (status %d)", e.StatusCode)
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
