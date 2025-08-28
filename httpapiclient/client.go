package httpapiclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"time"
)

type Client struct {
	baseURL    *url.URL
	httpClient *http.Client
}

// NewDefaultClient creates a new API client with the default http.DefaultClient
func NewDefaultClient(baseURL string) (*Client, error) {
	return NewClient(baseURL, http.DefaultClient)
}

// NewClient creates a new API client.
func NewClient(baseURL string, httpClient *http.Client) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 15 * time.Second}
	}
	return &Client{baseURL: u, httpClient: httpClient}, nil
}

// --- Public API Methods ---

func (c *Client) ListWebsites(ctx context.Context) (WebsiteListDTO, error) {
	var result WebsiteListDTO
	if err := c.doRequest(ctx, http.MethodGet, "/api/websites", nil, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *Client) CreateWebsite(ctx context.Context, dto WebsiteCreateDTO) (*WebsiteDTO, error) {
	var result WebsiteDTO
	if err := c.doRequest(ctx, http.MethodPost, "/api/websites", dto, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) UpdateWebsite(ctx context.Context, name string, dto WebsiteUpdateDTO) (*WebsiteDTO, error) {
	var result WebsiteDTO
	endpoint := path.Join("/api/websites", name)
	if err := c.doRequest(ctx, http.MethodPut, endpoint, dto, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

func (c *Client) DeleteWebsite(ctx context.Context, name string) error {
	endpoint := path.Join("/api/websites", name)
	return c.doRequest(ctx, http.MethodDelete, endpoint, nil, nil)
}

// --- Internal Helpers ---
func (c *Client) doRequest(ctx context.Context, method, endpoint string, body any, out any) error {
	u := *c.baseURL
	u.Path = path.Join(c.baseURL.Path, endpoint)

	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		buf = bytes.NewBuffer(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), buf)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b))
	}

	if out != nil {
		dec := json.NewDecoder(resp.Body)
		if err := dec.Decode(out); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}
	return nil
}
