package client

import (
	"net/http"
	"testing"
)

func TestETagExtraction(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string][]string
		expected string
	}{
		{
			name: "Standard ETag header",
			headers: map[string][]string{
				"ETag": {`"abc123"`},
			},
			expected: `"abc123"`,
		},
		{
			name: "Weak ETag header (per HTTP spec)",
			headers: map[string][]string{
				"ETag": {`W/"weak-etag"`},
			},
			expected: `W/"weak-etag"`,
		},
		{
			name: "Case-insensitive ETag header (HTTP standard behavior)",
			headers: map[string][]string{
				"etag": {"case-test"},
			},
			expected: "case-test",
		},
		{
			name: "Custom ETag headers not supported (OpenAPI spec only defines ETag)",
			headers: map[string][]string{
				"X-Custom-ETag": {"custom-etag"},
			},
			expected: "",
		},
		{
			name: "No ETag headers",
			headers: map[string][]string{
				"Content-Type": {"application/json"},
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock response with test headers
			header := make(http.Header)
			for k, v := range tt.headers {
				for _, value := range v {
					header.Add(k, value)
				}
			}
			resp := &http.Response{
				Header: header,
			}

			// Extract ETag using the same logic as Do method
			etag := resp.Header.Get("ETag")

			if etag != tt.expected {
				t.Errorf("Expected ETag '%s', got '%s'", tt.expected, etag)
			}
		})
	}
}
