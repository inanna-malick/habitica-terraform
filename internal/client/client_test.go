package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientRequestHeaders(t *testing.T) {
	var capturedHeaders http.Header

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer server.Close()

	client := New(Config{
		UserID:         "test-user-123",
		APIKey:         "test-key-456",
		ClientAuthorID: "test-author-789",
		ClientAppName:  "TestApp",
		BaseURL:        server.URL,
	})

	_, err := client.Get(context.Background(), "/test")
	require.NoError(t, err)

	assert.Equal(t, "test-user-123", capturedHeaders.Get("x-api-user"))
	assert.Equal(t, "test-key-456", capturedHeaders.Get("x-api-key"))
	assert.Equal(t, "test-author-789-TestApp", capturedHeaders.Get("x-client"))
	assert.Equal(t, "application/json", capturedHeaders.Get("Content-Type"))
}

func TestClientRateLimitRespected(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		// Return rate limit headers
		w.Header().Set("X-RateLimit-Remaining", "10")
		w.Header().Set("X-RateLimit-Reset", time.Now().Add(1*time.Second).Format(time.RFC3339))
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer server.Close()

	client := New(Config{
		UserID:          "test-user",
		APIKey:          "test-key",
		ClientAuthorID:  "test-author",
		RateLimitBuffer: 5,
		BaseURL:         server.URL,
	})

	// Call should succeed and update rate limit state
	_, err := client.Get(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)
}

func TestClientRetryOn429(t *testing.T){
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte(`{"success":false,"error":"Rate limited"}`))
		} else {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"success":true,"data":{}}`))
		}
	}))
	defer server.Close()

	client := New(Config{
		UserID:         "test-user",
		APIKey:         "test-key",
		ClientAuthorID: "test-author",
		MaxRetries:     5,
		BaseRetryDelay: 10 * time.Millisecond,
		BaseURL:        server.URL,
	})

	_, err := client.Get(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, 3, attempts)
}

func TestClientMaxRetriesExceeded(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"success":false,"error":"Rate limited"}`))
	}))
	defer server.Close()

	client := New(Config{
		UserID:         "test-user",
		APIKey:         "test-key",
		ClientAuthorID: "test-author",
		MaxRetries:     3,
		BaseRetryDelay: 10 * time.Millisecond,
		BaseURL:        server.URL,
	})

	_, err := client.Get(context.Background(), "/test")
	require.Error(t, err)
	assert.Equal(t, 4, attempts) // initial + 3 retries
}

func TestClient4xxErrorNoRetry(t *testing.T) {
	attempts := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"Bad request"}`))
	}))
	defer server.Close()

	client := New(Config{
		UserID:         "test-user",
		APIKey:         "test-key",
		ClientAuthorID: "test-author",
		BaseURL:        server.URL,
	})

	_, err := client.Get(context.Background(), "/test")
	require.Error(t, err)
	assert.Equal(t, 1, attempts) // No retries for 4xx errors
}

func TestClientTaskCachePopulation(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		if r.URL.Path == "/tasks/user" {
			// Bulk fetch endpoint
			w.Write([]byte(`{"success":true,"data":[
				{"id":"task-1","type":"habit","text":"Exercise"},
				{"id":"task-2","type":"daily","text":"Meditate"}
			]}`))
		} else {
			w.Write([]byte(`{"success":true,"data":{"id":"task-1","type":"habit","text":"Exercise"}}`))
		}
	}))
	defer server.Close()

	client := New(Config{
		UserID:         "test-user",
		APIKey:         "test-key",
		ClientAuthorID: "test-author",
		BaseURL:        server.URL,
	})

	// First GetTask should trigger bulk fetch
	task1, err := client.GetTask(context.Background(), "task-1")
	require.NoError(t, err)
	assert.Equal(t, "task-1", task1.ID)
	assert.Equal(t, 1, callCount) // Bulk fetch

	// Second GetTask should use cache
	task2, err := client.GetTask(context.Background(), "task-2")
	require.NoError(t, err)
	assert.Equal(t, "task-2", task2.ID)
	assert.Equal(t, 1, callCount) // Still 1, used cache
}

func TestClientTagCachePopulation(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"data":[
			{"id":"tag-1","name":"work"},
			{"id":"tag-2","name":"exercise"}
		]}`))
	}))
	defer server.Close()

	client := New(Config{
		UserID:         "test-user",
		APIKey:         "test-key",
		ClientAuthorID: "test-author",
		BaseURL:        server.URL,
	})

	// First GetTag should trigger bulk fetch
	tag1, err := client.GetTag(context.Background(), "tag-1")
	require.NoError(t, err)
	assert.Equal(t, "tag-1", tag1.ID)
	assert.Equal(t, 1, callCount)

	// Second GetTag should use cache
	tag2, err := client.GetTag(context.Background(), "tag-2")
	require.NoError(t, err)
	assert.Equal(t, "tag-2", tag2.ID)
	assert.Equal(t, 1, callCount) // Still 1, used cache
}

func TestClientCacheInvalidation(t *testing.T) {
	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.WriteHeader(http.StatusOK)

		switch r.Method {
		case http.MethodGet:
			w.Write([]byte(`{"success":true,"data":[{"id":"task-1","type":"habit","text":"Exercise"}]}`))
		case http.MethodPost:
			w.Write([]byte(`{"success":true,"data":{"id":"task-new","type":"habit","text":"New Task"}}`))
		case http.MethodPut:
			w.Write([]byte(`{"success":true,"data":{"id":"task-1","type":"habit","text":"Updated"}}`))
		case http.MethodDelete:
			w.Write([]byte(`{"success":true,"data":{}}`))
		}
	}))
	defer server.Close()

	client := New(Config{
		UserID:         "test-user",
		APIKey:         "test-key",
		ClientAuthorID: "test-author",
		BaseURL:        server.URL,
	})

	// Populate cache
	_, err := client.GetTask(context.Background(), "task-1")
	require.NoError(t, err)
	assert.Equal(t, 1, callCount)

	// Create should invalidate cache
	_, err = client.CreateTask(context.Background(), &Task{Type: "habit", Text: "New"})
	require.NoError(t, err)
	assert.Equal(t, 2, callCount)

	// Next GetTask should re-fetch (cache was invalidated)
	_, err = client.GetTask(context.Background(), "task-1")
	require.NoError(t, err)
	assert.Equal(t, 3, callCount) // Cache miss, new fetch

	// Update should also invalidate
	_, err = client.UpdateTask(context.Background(), "task-1", &Task{Text: "Updated"})
	require.NoError(t, err)

	// Delete should also invalidate
	err = client.DeleteTask(context.Background(), "task-1")
	require.NoError(t, err)
}

func TestClientJSONMarshaling(t *testing.T) {
	var capturedBody map[string]interface{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			var body map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&body)
			require.NoError(t, err)
			capturedBody = body
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"data":{"id":"new-task"}}`))
	}))
	defer server.Close()

	client := New(Config{
		UserID:         "test-user",
		APIKey:         "test-key",
		ClientAuthorID: "test-author",
		BaseURL:        server.URL,
	})

	upPtr := true
	_, err := client.CreateTask(context.Background(), &Task{
		Type:     "habit",
		Text:     "Test Task",
		Priority: 1.5,
		Up:       &upPtr,
	})
	require.NoError(t, err)

	assert.Equal(t, "habit", capturedBody["type"])
	assert.Equal(t, "Test Task", capturedBody["text"])
	assert.Equal(t, 1.5, capturedBody["priority"])
	assert.Equal(t, true, capturedBody["up"])
}

func TestClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success":true,"data":{}}`))
	}))
	defer server.Close()

	client := New(Config{
		UserID:         "test-user",
		APIKey:         "test-key",
		ClientAuthorID: "test-author",
		BaseURL:        server.URL,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, "/test")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "context")
}

func TestClientErrorResponseParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"success":false,"error":"ValidationError","message":"Invalid input"}`))
	}))
	defer server.Close()

	client := New(Config{
		UserID:         "test-user",
		APIKey:         "test-key",
		ClientAuthorID: "test-author",
		BaseURL:        server.URL,
	})

	_, err := client.Get(context.Background(), "/test")
	require.Error(t, err)
	// Error should contain either the status code or message
	assert.Contains(t, err.Error(), "400")
}
