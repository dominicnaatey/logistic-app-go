package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// UpstashClient wraps the Upstash REST API
type UpstashClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewUpstashClient creates a new Upstash REST API client
func NewUpstashClient(restURL, token string) (*UpstashClient, error) {
	if restURL == "" || token == "" {
		return nil, fmt.Errorf("Upstash REST URL and token are required")
	}

	client := &UpstashClient{
		baseURL: strings.TrimSuffix(restURL, "/"),
		token:   token,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping Upstash: %w", err)
	}

	log.Println("✓ Connected to Upstash Redis (REST API)")
	return client, nil
}

// Ping tests the connection to Upstash
func (c *UpstashClient) Ping(ctx context.Context) error {
	_, err := c.execute(ctx, "PING")
	return err
}

// Get retrieves a value from Upstash
func (c *UpstashClient) Get(ctx context.Context, key string) (string, error) {
	result, err := c.execute(ctx, "GET", key)
	if err != nil {
		return "", err
	}

	// Upstash returns null for missing keys
	if result == nil {
		return "", fmt.Errorf("key not found")
	}

	value, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("unexpected type: %T", result)
	}

	return value, nil
}

// Set stores a value in Upstash with expiration
func (c *UpstashClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if expiration > 0 {
		// Use SETEX for expiration
		seconds := int(expiration.Seconds())
		_, err := c.execute(ctx, "SETEX", key, seconds, value)
		return err
	}

	// Use SET without expiration
	_, err := c.execute(ctx, "SET", key, value)
	return err
}

// Del deletes a key from Upstash
func (c *UpstashClient) Del(ctx context.Context, keys ...string) error {
	args := make([]interface{}, len(keys))
	for i, key := range keys {
		args[i] = key
	}
	_, err := c.execute(ctx, "DEL", args...)
	return err
}

// Incr increments a counter
func (c *UpstashClient) Incr(ctx context.Context, key string) (int64, error) {
	result, err := c.execute(ctx, "INCR", key)
	if err != nil {
		return 0, err
	}

	// Upstash returns number as float64 in JSON
	switch v := result.(type) {
	case float64:
		return int64(v), nil
	case int64:
		return v, nil
	default:
		return 0, fmt.Errorf("unexpected type: %T", result)
	}
}

// Expire sets expiration on a key
func (c *UpstashClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	seconds := int(expiration.Seconds())
	_, err := c.execute(ctx, "EXPIRE", key, seconds)
	return err
}

// Publish publishes a message to a channel
func (c *UpstashClient) Publish(ctx context.Context, channel string, message interface{}) error {
	_, err := c.execute(ctx, "PUBLISH", channel, message)
	return err
}

// execute performs an HTTP request to the Upstash REST API
func (c *UpstashClient) execute(ctx context.Context, command string, args ...interface{}) (interface{}, error) {
	// Build command array
	cmdArray := []interface{}{command}
	cmdArray = append(cmdArray, args...)

	// Marshal to JSON
	body, err := json.Marshal(cmdArray)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal command: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	// Execute request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var response struct {
		Result interface{} `json:"result"`
		Error  string      `json:"error,omitempty"`
	}

	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("Redis error: %s", response.Error)
	}

	return response.Result, nil
}

// Close closes the HTTP client (no-op for REST API)
func (c *UpstashClient) Close() error {
	return nil
}
