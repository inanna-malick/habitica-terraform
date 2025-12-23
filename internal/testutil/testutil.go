package testutil

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/inannamalick/terraform-provider-habitica/internal/client"
)

// NewMockHabiticaServer creates an httptest.Server with custom route handlers
func NewMockHabiticaServer(t *testing.T, handlers map[string]http.HandlerFunc) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	for pattern, handler := range handlers {
		mux.HandleFunc(pattern, handler)
	}

	return httptest.NewServer(mux)
}

// MockTaskResponse returns JSON bytes for a Task wrapped in APIResponse
func MockTaskResponse(task *client.Task) []byte {
	resp := client.APIResponse[*client.Task]{
		Success: true,
		Data:    task,
	}
	bytes, _ := json.Marshal(resp)
	return bytes
}

// MockTasksResponse returns JSON bytes for multiple Tasks wrapped in APIResponse
func MockTasksResponse(tasks []client.Task) []byte {
	resp := client.APIResponse[[]client.Task]{
		Success: true,
		Data:    tasks,
	}
	bytes, _ := json.Marshal(resp)
	return bytes
}

// MockTagResponse returns JSON bytes for a Tag wrapped in APIResponse
func MockTagResponse(tag *client.Tag) []byte {
	resp := client.APIResponse[*client.Tag]{
		Success: true,
		Data:    tag,
	}
	bytes, _ := json.Marshal(resp)
	return bytes
}

// MockTagsResponse returns JSON bytes for multiple Tags wrapped in APIResponse
func MockTagsResponse(tags []client.Tag) []byte {
	resp := client.APIResponse[[]client.Tag]{
		Success: true,
		Data:    tags,
	}
	bytes, _ := json.Marshal(resp)
	return bytes
}

// MockWebhookResponse returns JSON bytes for a Webhook wrapped in APIResponse
func MockWebhookResponse(webhook *client.Webhook) []byte {
	resp := client.APIResponse[*client.Webhook]{
		Success: true,
		Data:    webhook,
	}
	bytes, _ := json.Marshal(resp)
	return bytes
}

// MockWebhooksResponse returns JSON bytes for multiple Webhooks wrapped in APIResponse
func MockWebhooksResponse(webhooks []client.Webhook) []byte {
	resp := client.APIResponse[[]client.Webhook]{
		Success: true,
		Data:    webhooks,
	}
	bytes, _ := json.Marshal(resp)
	return bytes
}

// MockErrorResponse returns JSON bytes for an API error
func MockErrorResponse(statusCode int, message string) []byte {
	resp := client.APIResponse[interface{}]{
		Success: false,
		Error:   http.StatusText(statusCode),
		Message: message,
	}
	bytes, _ := json.Marshal(resp)
	return bytes
}

// MockRateLimitHeaders returns HTTP headers with rate limit information
func MockRateLimitHeaders(remaining int, resetTime time.Time) http.Header {
	headers := http.Header{}
	headers.Set("X-RateLimit-Remaining", string(rune(remaining+'0')))
	headers.Set("X-RateLimit-Reset", resetTime.Format(time.RFC3339))
	return headers
}

// NewTestClient creates a client.Client pointing to the test server URL
func NewTestClient(serverURL string) *client.Client {
	return client.New(client.Config{
		UserID:          "test-user-id",
		APIKey:          "test-api-key",
		ClientAuthorID:  "test-client-author-id",
		ClientAppName:   "TestApp",
		RateLimitBuffer: 5,
		BaseRetryDelay:  10 * time.Millisecond, // Faster retries for tests
		BaseURL:         serverURL,
	})
}

// BoolPtr returns a pointer to a bool value
func BoolPtr(b bool) *bool {
	return &b
}

// TimePtr returns a pointer to a time.Time value
func TimePtr(t time.Time) *time.Time {
	return &t
}
