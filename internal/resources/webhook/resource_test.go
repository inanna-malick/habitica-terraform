package webhook

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

// TestWebhookClientCreate validates webhook creation via client
func TestWebhookClientCreate(t *testing.T) {
	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/user/webhook": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)

			var webhook client.Webhook
			err := json.NewDecoder(r.Body).Decode(&webhook)
			require.NoError(t, err)

			assert.Equal(t, "https://example.com/webhook", webhook.URL)
			assert.Equal(t, "test-webhook", webhook.Label)
			assert.Equal(t, "taskActivity", webhook.Type)
			assert.True(t, webhook.Enabled)

			created := &client.Webhook{
				ID:      "webhook-123",
				URL:     webhook.URL,
				Label:   webhook.Label,
				Type:    webhook.Type,
				Enabled: webhook.Enabled,
				Options: webhook.Options,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    created,
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)
	webhook, err := c.CreateWebhook(context.Background(), &client.Webhook{
		URL:     "https://example.com/webhook",
		Label:   "test-webhook",
		Type:    "taskActivity",
		Enabled: true,
		Options: client.WebhookOptions{
			Created: true,
			Updated: false,
		},
	})

	require.NoError(t, err)
	assert.Equal(t, "webhook-123", webhook.ID)
	assert.Equal(t, "https://example.com/webhook", webhook.URL)
	assert.Equal(t, "test-webhook", webhook.Label)
	assert.Equal(t, "taskActivity", webhook.Type)
	assert.True(t, webhook.Enabled)
}

// TestWebhookClientRead validates webhook read via client
func TestWebhookClientRead(t *testing.T) {
	testWebhook := testutil.TestWebhook1

	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/user/webhook": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    []client.Webhook{testWebhook},
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)
	webhook, err := c.GetWebhook(context.Background(), testWebhook.ID)

	require.NoError(t, err)
	assert.Equal(t, testWebhook.ID, webhook.ID)
	assert.Equal(t, testWebhook.URL, webhook.URL)
	assert.Equal(t, testWebhook.Label, webhook.Label)
	assert.Equal(t, testWebhook.Type, webhook.Type)
}

// TestWebhookClientUpdate validates webhook update via client
func TestWebhookClientUpdate(t *testing.T) {
	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/user/webhook/webhook-123": func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)

			var webhook client.Webhook
			err := json.NewDecoder(r.Body).Decode(&webhook)
			require.NoError(t, err)

			assert.Equal(t, "https://example.com/updated", webhook.URL)
			assert.Equal(t, "updated-label", webhook.Label)

			updated := &client.Webhook{
				ID:      "webhook-123",
				URL:     webhook.URL,
				Label:   webhook.Label,
				Type:    webhook.Type,
				Enabled: webhook.Enabled,
				Options: webhook.Options,
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    updated,
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)
	webhook, err := c.UpdateWebhook(context.Background(), "webhook-123", &client.Webhook{
		URL:     "https://example.com/updated",
		Label:   "updated-label",
		Type:    "taskActivity",
		Enabled: true,
	})

	require.NoError(t, err)
	assert.Equal(t, "webhook-123", webhook.ID)
	assert.Equal(t, "https://example.com/updated", webhook.URL)
	assert.Equal(t, "updated-label", webhook.Label)
}

// TestWebhookClientDelete validates webhook deletion via client
func TestWebhookClientDelete(t *testing.T) {
	deleteCallCount := 0

	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/user/webhook/webhook-123": func(w http.ResponseWriter, r *http.Request) {
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
	err := c.DeleteWebhook(context.Background(), "webhook-123")

	require.NoError(t, err)
	assert.Equal(t, 1, deleteCallCount, "Delete endpoint should be called exactly once")
}

// TestWebhookGetWebhookFiltering validates that GetWebhook filters list results
func TestWebhookGetWebhookFiltering(t *testing.T) {
	allWebhooks := []client.Webhook{
		{
			ID:    "webhook-1",
			URL:   "https://example.com/1",
			Label: "webhook-one",
			Type:  "taskActivity",
		},
		{
			ID:    "webhook-2",
			URL:   "https://example.com/2",
			Label: "webhook-two",
			Type:  "userActivity",
		},
	}

	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/user/webhook": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    allWebhooks,
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)

	// Get webhook-2
	webhook, err := c.GetWebhook(context.Background(), "webhook-2")
	require.NoError(t, err)
	assert.Equal(t, "webhook-2", webhook.ID)
	assert.Equal(t, "webhook-two", webhook.Label)
}

// TestWebhookGetWebhookNotFound validates error handling for missing webhook
func TestWebhookGetWebhookNotFound(t *testing.T) {
	server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
		"/user/webhook": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"data":    []client.Webhook{},
			})
		},
	})
	defer server.Close()

	c := testutil.NewTestClient(server.URL)
	_, err := c.GetWebhook(context.Background(), "nonexistent-webhook")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestWebhookOptions validates webhook options handling
func TestWebhookOptions(t *testing.T) {
	tests := []struct {
		name    string
		options client.WebhookOptions
	}{
		{
			name: "all options enabled",
			options: client.WebhookOptions{
				Created:         true,
				Updated:         true,
				Deleted:         true,
				Scored:          true,
				ChecklistScored: true,
			},
		},
		{
			name: "all options disabled",
			options: client.WebhookOptions{
				Created:         false,
				Updated:         false,
				Deleted:         false,
				Scored:          false,
				ChecklistScored: false,
			},
		},
		{
			name: "partial options",
			options: client.WebhookOptions{
				Created: true,
				Scored:  true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
				"/user/webhook": func(w http.ResponseWriter, r *http.Request) {
					var webhook client.Webhook
					json.NewDecoder(r.Body).Decode(&webhook)

					assert.Equal(t, tt.options.Created, webhook.Options.Created)
					assert.Equal(t, tt.options.Updated, webhook.Options.Updated)
					assert.Equal(t, tt.options.Deleted, webhook.Options.Deleted)
					assert.Equal(t, tt.options.Scored, webhook.Options.Scored)
					assert.Equal(t, tt.options.ChecklistScored, webhook.Options.ChecklistScored)

					created := &client.Webhook{
						ID:      "webhook-123",
						URL:     webhook.URL,
						Type:    webhook.Type,
						Enabled: webhook.Enabled,
						Options: webhook.Options,
					}

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": true,
						"data":    created,
					})
				},
			})
			defer server.Close()

			c := testutil.NewTestClient(server.URL)
			webhook, err := c.CreateWebhook(context.Background(), &client.Webhook{
				URL:     "https://example.com/test",
				Type:    "taskActivity",
				Enabled: true,
				Options: tt.options,
			})

			require.NoError(t, err)
			assert.Equal(t, tt.options, webhook.Options)
		})
	}
}

// TestWebhookTypes validates different webhook type values
func TestWebhookTypes(t *testing.T) {
	types := []string{
		"taskActivity",
		"userActivity",
		"questActivity",
		"groupChatReceived",
	}

	for _, webhookType := range types {
		t.Run(webhookType, func(t *testing.T) {
			server := testutil.NewMockHabiticaServer(t, map[string]http.HandlerFunc{
				"/user/webhook": func(w http.ResponseWriter, r *http.Request) {
					var webhook client.Webhook
					json.NewDecoder(r.Body).Decode(&webhook)

					assert.Equal(t, webhookType, webhook.Type)

					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"success": true,
						"data": client.Webhook{
							ID:      "webhook-123",
							URL:     webhook.URL,
							Type:    webhook.Type,
							Enabled: webhook.Enabled,
						},
					})
				},
			})
			defer server.Close()

			c := testutil.NewTestClient(server.URL)
			webhook, err := c.CreateWebhook(context.Background(), &client.Webhook{
				URL:     "https://example.com/test",
				Type:    webhookType,
				Enabled: true,
			})

			require.NoError(t, err)
			assert.Equal(t, webhookType, webhook.Type)
		})
	}
}
