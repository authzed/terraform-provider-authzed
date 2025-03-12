package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type APIError struct {
	StatusCode int
	Message string
	Body []byte
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("API error (status %d): %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("API error (status %d)", e.StatusCode)
}

// NewAPIError creates a new APIError from an HTTP response
func NewAPIError(resp *http.Response) *APIError {
	body, _ := io.ReadAll(resp.Body)

	var errMsg string
	// Try to parse as JSON if possible
	var jsonErr map[string]interface{}
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
		Body:       body,
	}
}
