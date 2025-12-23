package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

const (
	DefaultBaseURL      = "https://habitica.com/api/v3"
	DefaultRateLimitBuf = 5
	DefaultMaxRetries   = 5
	DefaultRetryDelay   = 2 * time.Second
)

// Client is an HTTP client for the Habitica API.
type Client struct {
	baseURL    string
	userID     string
	apiKey     string
	clientID   string // x-client header value
	httpClient *http.Client

	// Rate limiting
	rateLimitBuffer int
	maxRetries      int
	baseRetryDelay  time.Duration

	mu                 sync.Mutex
	rateLimitRemaining int
	rateLimitReset     time.Time

	// Caches for bulk fetching
	taskCache   map[string]*Task
	taskCacheMu sync.RWMutex
	tagCache    map[string]*Tag
	tagCacheMu  sync.RWMutex
}

// Config holds configuration for creating a new Client.
type Config struct {
	UserID          string
	APIKey          string
	ClientAuthorID  string
	ClientAppName   string
	RateLimitBuffer int
	MaxRetries      int
	BaseRetryDelay  time.Duration
}

// New creates a new Habitica API client.
func New(cfg Config) *Client {
	appName := cfg.ClientAppName
	if appName == "" {
		appName = "TerraformHabitica"
	}

	rateLimitBuffer := cfg.RateLimitBuffer
	if rateLimitBuffer <= 0 {
		rateLimitBuffer = DefaultRateLimitBuf
	}

	maxRetries := cfg.MaxRetries
	if maxRetries <= 0 {
		maxRetries = DefaultMaxRetries
	}

	baseRetryDelay := cfg.BaseRetryDelay
	if baseRetryDelay <= 0 {
		baseRetryDelay = DefaultRetryDelay
	}

	return &Client{
		baseURL:            DefaultBaseURL,
		userID:             cfg.UserID,
		apiKey:             cfg.APIKey,
		clientID:           fmt.Sprintf("%s-%s", cfg.ClientAuthorID, appName),
		httpClient:         &http.Client{Timeout: 30 * time.Second},
		rateLimitBuffer:    rateLimitBuffer,
		maxRetries:         maxRetries,
		baseRetryDelay:     baseRetryDelay,
		rateLimitRemaining: 30, // Start optimistic
	}
}

// do executes an HTTP request with rate limiting and retry logic.
func (c *Client) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			delay := c.baseRetryDelay * time.Duration(1<<(attempt-1)) // Exponential backoff
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(delay):
			}
		}

		// Check rate limit before making request
		c.mu.Lock()
		if c.rateLimitRemaining < c.rateLimitBuffer && time.Now().Before(c.rateLimitReset) {
			waitTime := time.Until(c.rateLimitReset)
			c.mu.Unlock()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(waitTime):
			}
		} else {
			c.mu.Unlock()
		}

		// Recreate body reader for retries
		if body != nil {
			jsonBody, _ := json.Marshal(body)
			bodyReader = bytes.NewReader(jsonBody)
		}

		req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
		if err != nil {
			return nil, fmt.Errorf("creating request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("x-api-user", c.userID)
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("x-client", c.clientID)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("executing request: %w", err)
			continue
		}

		// Update rate limit info
		c.updateRateLimits(resp)

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			lastErr = fmt.Errorf("reading response body: %w", err)
			continue
		}

		// Handle rate limiting
		if resp.StatusCode == http.StatusTooManyRequests {
			lastErr = fmt.Errorf("rate limited (429)")
			continue
		}

		// Handle other errors
		if resp.StatusCode >= 400 {
			var apiResp APIResponse[any]
			if err := json.Unmarshal(respBody, &apiResp); err == nil && apiResp.Message != "" {
				return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, apiResp.Message)
			}
			return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
		}

		return respBody, nil
	}

	return nil, fmt.Errorf("max retries exceeded: %w", lastErr)
}

func (c *Client) updateRateLimits(resp *http.Response) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if remaining := resp.Header.Get("X-RateLimit-Remaining"); remaining != "" {
		if val, err := strconv.Atoi(remaining); err == nil {
			c.rateLimitRemaining = val
		}
	}

	if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
		if ts, err := strconv.ParseInt(reset, 10, 64); err == nil {
			c.rateLimitReset = time.Unix(ts, 0)
		}
	}
}

// Get performs a GET request.
func (c *Client) Get(ctx context.Context, path string) ([]byte, error) {
	return c.do(ctx, http.MethodGet, path, nil)
}

// Post performs a POST request.
func (c *Client) Post(ctx context.Context, path string, body any) ([]byte, error) {
	return c.do(ctx, http.MethodPost, path, body)
}

// Put performs a PUT request.
func (c *Client) Put(ctx context.Context, path string, body any) ([]byte, error) {
	return c.do(ctx, http.MethodPut, path, body)
}

// Delete performs a DELETE request.
func (c *Client) Delete(ctx context.Context, path string) ([]byte, error) {
	return c.do(ctx, http.MethodDelete, path, nil)
}

// Tag operations

// CreateTag creates a new tag.
func (c *Client) CreateTag(ctx context.Context, name string) (*Tag, error) {
	body := map[string]string{"name": name}
	resp, err := c.Post(ctx, "/tags", body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Tag]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	c.invalidateTagCache()
	return &apiResp.Data, nil
}

// GetTag retrieves a tag by ID, using cache if available.
func (c *Client) GetTag(ctx context.Context, id string) (*Tag, error) {
	// Check cache first
	c.tagCacheMu.RLock()
	if c.tagCache != nil {
		if tag, ok := c.tagCache[id]; ok {
			c.tagCacheMu.RUnlock()
			return tag, nil
		}
	}
	c.tagCacheMu.RUnlock()

	// Cache miss - populate cache with all tags
	if err := c.populateTagCache(ctx); err != nil {
		return nil, err
	}

	// Try cache again
	c.tagCacheMu.RLock()
	defer c.tagCacheMu.RUnlock()
	if tag, ok := c.tagCache[id]; ok {
		return tag, nil
	}

	return nil, fmt.Errorf("tag not found: %s", id)
}

// populateTagCache fetches all tags and caches them.
func (c *Client) populateTagCache(ctx context.Context) error {
	c.tagCacheMu.Lock()
	defer c.tagCacheMu.Unlock()

	// Already populated by another goroutine
	if c.tagCache != nil {
		return nil
	}

	resp, err := c.Get(ctx, "/tags")
	if err != nil {
		return err
	}

	var apiResp APIResponse[[]Tag]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}

	c.tagCache = make(map[string]*Tag)
	for i := range apiResp.Data {
		c.tagCache[apiResp.Data[i].ID] = &apiResp.Data[i]
	}

	return nil
}

// UpdateTag updates a tag.
func (c *Client) UpdateTag(ctx context.Context, id, name string) (*Tag, error) {
	body := map[string]string{"name": name}
	resp, err := c.Put(ctx, "/tags/"+id, body)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Tag]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	c.invalidateTagCache()
	return &apiResp.Data, nil
}

// DeleteTag deletes a tag.
func (c *Client) DeleteTag(ctx context.Context, id string) error {
	_, err := c.Delete(ctx, "/tags/"+id)
	if err == nil {
		c.invalidateTagCache()
	}
	return err
}

func (c *Client) invalidateTagCache() {
	c.tagCacheMu.Lock()
	c.tagCache = nil
	c.tagCacheMu.Unlock()
}

// Task operations

// CreateTask creates a new task.
func (c *Client) CreateTask(ctx context.Context, task *Task) (*Task, error) {
	resp, err := c.Post(ctx, "/tasks/user", task)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Task]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	c.invalidateTaskCache()
	return &apiResp.Data, nil
}

// GetTask retrieves a task by ID, using cache if available.
func (c *Client) GetTask(ctx context.Context, id string) (*Task, error) {
	// Check cache first
	c.taskCacheMu.RLock()
	if c.taskCache != nil {
		if task, ok := c.taskCache[id]; ok {
			c.taskCacheMu.RUnlock()
			return task, nil
		}
	}
	c.taskCacheMu.RUnlock()

	// Cache miss - populate cache with all tasks
	if err := c.populateTaskCache(ctx); err != nil {
		return nil, err
	}

	// Try cache again
	c.taskCacheMu.RLock()
	defer c.taskCacheMu.RUnlock()
	if task, ok := c.taskCache[id]; ok {
		return task, nil
	}

	return nil, fmt.Errorf("task not found: %s", id)
}

// populateTaskCache fetches all tasks and caches them.
func (c *Client) populateTaskCache(ctx context.Context) error {
	c.taskCacheMu.Lock()
	defer c.taskCacheMu.Unlock()

	// Already populated by another goroutine
	if c.taskCache != nil {
		return nil
	}

	resp, err := c.Get(ctx, "/tasks/user")
	if err != nil {
		return err
	}

	var apiResp APIResponse[[]Task]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return fmt.Errorf("unmarshaling response: %w", err)
	}

	c.taskCache = make(map[string]*Task)
	for i := range apiResp.Data {
		c.taskCache[apiResp.Data[i].ID] = &apiResp.Data[i]
	}

	return nil
}

// UpdateTask updates a task.
func (c *Client) UpdateTask(ctx context.Context, id string, task *Task) (*Task, error) {
	resp, err := c.Put(ctx, "/tasks/"+id, task)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Task]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	c.invalidateTaskCache()
	return &apiResp.Data, nil
}

// DeleteTask deletes a task.
func (c *Client) DeleteTask(ctx context.Context, id string) error {
	_, err := c.Delete(ctx, "/tasks/"+id)
	if err == nil {
		c.invalidateTaskCache()
	}
	return err
}

func (c *Client) invalidateTaskCache() {
	c.taskCacheMu.Lock()
	c.taskCache = nil
	c.taskCacheMu.Unlock()
}

// GetAllTasks retrieves all tasks for the user.
func (c *Client) GetAllTasks(ctx context.Context) ([]Task, error) {
	resp, err := c.Get(ctx, "/tasks/user")
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[[]Task]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return apiResp.Data, nil
}

// GetAllTags retrieves all tags for the user.
func (c *Client) GetAllTags(ctx context.Context) ([]Tag, error) {
	resp, err := c.Get(ctx, "/tags")
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[[]Tag]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return apiResp.Data, nil
}

// Webhook operations

// CreateWebhook creates a new webhook.
func (c *Client) CreateWebhook(ctx context.Context, webhook *Webhook) (*Webhook, error) {
	resp, err := c.Post(ctx, "/user/webhook", webhook)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Webhook]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &apiResp.Data, nil
}

// GetWebhooks retrieves all webhooks.
func (c *Client) GetWebhooks(ctx context.Context) ([]Webhook, error) {
	resp, err := c.Get(ctx, "/user/webhook")
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[[]Webhook]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return apiResp.Data, nil
}

// GetWebhook retrieves a webhook by ID.
func (c *Client) GetWebhook(ctx context.Context, id string) (*Webhook, error) {
	webhooks, err := c.GetWebhooks(ctx)
	if err != nil {
		return nil, err
	}

	for _, wh := range webhooks {
		if wh.ID == id {
			return &wh, nil
		}
	}

	return nil, fmt.Errorf("webhook not found: %s", id)
}

// UpdateWebhook updates a webhook.
func (c *Client) UpdateWebhook(ctx context.Context, id string, webhook *Webhook) (*Webhook, error) {
	resp, err := c.Put(ctx, "/user/webhook/"+id, webhook)
	if err != nil {
		return nil, err
	}

	var apiResp APIResponse[Webhook]
	if err := json.Unmarshal(resp, &apiResp); err != nil {
		return nil, fmt.Errorf("unmarshaling response: %w", err)
	}

	return &apiResp.Data, nil
}

// DeleteWebhook deletes a webhook.
func (c *Client) DeleteWebhook(ctx context.Context, id string) error {
	_, err := c.Delete(ctx, "/user/webhook/"+id)
	return err
}
