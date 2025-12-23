package tag

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/inannamalick/terraform-provider-habitica/internal/client"
	"github.com/inannamalick/terraform-provider-habitica/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTagClientCreate validates tag creation via client
func TestTagClientCreate(t *testing.T) {
	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/tags": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)

			var req struct {
				Name string `json:"name"`
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			assert.Equal(t, "test-tag", req.Name)

			tag := &client.Tag{
				ID:   "tag-123",
				Name: "test-tag",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    tag,
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)
	tag, err := c.CreateTag(context.Background(), "test-tag")

	require.NoError(t, err)
	assert.Equal(t, "tag-123", tag.ID)
	assert.Equal(t, "test-tag", tag.Name)
}

// TestTagClientRead validates tag read via client (uses cache)
func TestTagClientRead(t *testing.T) {
	testTag := testutil.TestTag1

	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/tags": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    []client.Tag{testTag},
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)
	tag, err := c.GetTag(context.Background(), testTag.ID)

	require.NoError(t, err)
	assert.Equal(t, testTag.ID, tag.ID)
	assert.Equal(t, testTag.Name, tag.Name)
}

// TestTagClientUpdate validates tag update via client
func TestTagClientUpdate(t *testing.T) {
	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/tags/tag-123": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)

			var req struct {
				Name string `json:"name"`
			}
			err := json.NewDecoder(r.Body).Decode(&req)
			require.NoError(t, err)

			assert.Equal(t, "updated-tag", req.Name)

			tag := &client.Tag{
				ID:   "tag-123",
				Name: "updated-tag",
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    tag,
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)
	tag, err := c.UpdateTag(context.Background(), "tag-123", "updated-tag")

	require.NoError(t, err)
	assert.Equal(t, "tag-123", tag.ID)
	assert.Equal(t, "updated-tag", tag.Name)
}

// TestTagClientDelete validates tag deletion via client
func TestTagClientDelete(t *testing.T) {
	deleteCallCount := 0

	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/tags/tag-123": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodDelete, r.Method)
			deleteCallCount++

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    map[string]interface{}{},
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)
	err := c.DeleteTag(context.Background(), "tag-123")

	require.NoError(t, err)
	assert.Equal(t, 1, deleteCallCount, "Delete endpoint should be called exactly once")
}

// TestTagClientCacheInvalidation validates that tag cache is cleared on write operations
func TestTagClientCacheInvalidation(t *testing.T) {
	listCallCount := 0

	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/tags": func(w http.ResponseWriter, r *http.Request) {
			listCallCount++

			tags := []client.Tag{
				{ID: "tag-1", Name: "tag-one"},
				{ID: "tag-2", Name: "tag-two"},
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    tags,
			})
		},
		"/tags/tag-1": func(w http.ResponseWriter, r *http.Request) {
			// Update endpoint
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    client.Tag{ID: "tag-1", Name: "updated"},
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)

	// First read - populates cache
	_, err := c.GetTag(context.Background(), "tag-1")
	require.NoError(t, err)
	assert.Equal(t, 1, listCallCount)

	// Second read - uses cache
	_, err = c.GetTag(context.Background(), "tag-1")
	require.NoError(t, err)
	assert.Equal(t, 1, listCallCount, "Should use cached value")

	// Update - invalidates cache
	_, err = c.UpdateTag(context.Background(), "tag-1", "updated")
	require.NoError(t, err)

	// Third read - repopulates cache after invalidation
	_, err = c.GetTag(context.Background(), "tag-1")
	require.NoError(t, err)
	assert.Equal(t, 2, listCallCount, "Cache should be invalidated after update")
}

// TestTagNameValidation validates tag name handling
func TestTagNameValidation(t *testing.T) {
	tests := []struct {
		name     string
		tagName  string
		expected string
	}{
		{
			name:     "simple name",
			tagName:  "work",
			expected: "work",
		},
		{
			name:     "name with spaces",
			tagName:  "work project",
			expected: "work project",
		},
		{
			name:     "name with special chars",
			tagName:  "tier:foundation",
			expected: "tier:foundation",
		},
		{
			name:     "unicode name",
			tagName:  "🏋️ exercise",
			expected: "🏋️ exercise",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
				"/tags": func(w http.ResponseWriter, r *http.Request) {
					var req struct {
						Name string `json:"name"`
					}
					json.NewDecoder(r.Body).Decode(&req)

					assert.Equal(t, tt.expected, req.Name)

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": true,
						"data": client.Tag{
							ID:   "tag-123",
							Name: req.Name,
						},
					})
				},
			})
			defer server.Close()

			c := testutil.NewTestClient(server.URL)
			tag, err := c.CreateTag(context.Background(), tt.tagName)

			require.NoError(t, err)
			assert.Equal(t, tt.expected, tag.Name)
		})
	}
}
