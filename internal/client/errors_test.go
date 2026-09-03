package client

import (
	"bytes"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// responseWithBody builds a response the way NewAPIError expects to receive one.
func responseWithBody(t *testing.T, status int, body string) *HTTPResponseWrapper {
	t.Helper()
	target, err := url.Parse("https://api.example.com/materialize/mc-test")
	if err != nil {
		t.Fatal(err)
	}
	return &HTTPResponseWrapper{Response: &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Request:    &http.Request{Method: http.MethodPatch, URL: target},
	}}
}

func TestNewAPIErrorIncludesFieldDetails(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			// The generic top-level message is useless on its own; the
			// explanation of what to fix is in the errors array.
			name: "detail explains a generic message",
			body: `{"code":400,"message":"bad request","errors":[{"domain":"global","reason":"badRequest","message":"watchedPermissions[0] must match resource_type/resource_id#relation@subject_type/subject_id","locationType":"body"}]}`,
			want: "API error (status 400): bad request: watchedPermissions[0] must match resource_type/resource_id#relation@subject_type/subject_id",
		},
		{
			name: "several details are joined",
			body: `{"code":400,"message":"validation failed","errors":[{"message":"watchedPermissions must not be empty"},{"message":"replicas must be at least 0"}]}`,
			want: "API error (status 400): validation failed: watchedPermissions must not be empty; replicas must be at least 0",
		},
		{
			name: "detail alone when there is no summary",
			body: `{"code":400,"errors":[{"message":"watchedPermissions must not be empty"}]}`,
			want: "API error (status 400): watchedPermissions must not be empty",
		},
		{
			name: "a detail repeating the summary is not said twice",
			body: `{"code":404,"message":"server template \"mtsc-nope\" not found","errors":[{"message":"server template \"mtsc-nope\" not found"}]}`,
			want: `API error (status 404): server template "mtsc-nope" not found`,
		},
		{
			name: "empty details are skipped",
			body: `{"code":400,"message":"bad request","errors":[{"message":""},{"reason":"badRequest"}]}`,
			want: "API error (status 400): bad request",
		},
		{
			name: "unchanged when there are no details",
			body: `{"code":409,"message":"resource conflict"}`,
			want: "API error (status 409): resource conflict",
		},
		{
			name: "unchanged for a non-JSON body",
			body: `<html>gateway timeout</html>`,
			want: "API error (status 504)",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status := 400
			switch {
			case strings.Contains(tc.body, `"code":404`):
				status = 404
			case strings.Contains(tc.body, `"code":409`):
				status = 409
			case !strings.HasPrefix(tc.body, "{"):
				status = 504
			}
			got := NewAPIError(responseWithBody(t, status, tc.body)).Error()
			if got != tc.want {
				t.Errorf("Error() =\n  %q\nwant\n  %q", got, tc.want)
			}
		})
	}
}

func TestNewAPIErrorKeepsFGAMGuidance(t *testing.T) {
	// The 409 guidance keys off the message, so details must not hide it.
	body := `{"code":409,"message":"restricted API access configuration for permission system \"ps-test\" has changed","errors":[{"message":"please retry"}]}`
	got := NewAPIError(responseWithBody(t, http.StatusConflict, body)).Error()
	if !strings.Contains(got, "Fine-Grained Access Management") {
		t.Errorf("expected the FGAM guidance to survive, got: %s", got)
	}
	if !strings.Contains(got, "please retry") {
		t.Errorf("expected the detail to be included, got: %s", got)
	}
}
