package httpclient

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"moph-api-proxy/internal/config"
)

// APIClient wraps outbound HTTP calls with automatic token injection and retry.
type APIClient struct {
	app         string
	baseURL     *url.URL
	client      *http.Client
	tokenSource *TokenManager
}

// NewAPIClient builds an API client for a given base URL and application label.
func NewAPIClient(app string, base string, manager *TokenManager) (*APIClient, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("parse base URL: %w", err)
	}
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}
	client := &http.Client{
		Transport:     transport,
		Timeout:       30 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	return &APIClient{
		app:         app,
		baseURL:     parsed,
		client:      client,
		tokenSource: manager,
	}, nil
}

// Do forwards the request to the upstream service and retries once on 401.
func (c *APIClient) Do(ctx context.Context, method, path, rawQuery string, body []byte, headers http.Header) (*http.Response, error) {
	if c == nil {
		return nil, fmt.Errorf("API client is nil")
	}

	attempt := 0
	for {
		attempt++

		endpoint := c.baseURL.ResolveReference(&url.URL{Path: path, RawQuery: rawQuery})
		var bodyReader io.Reader
		if body != nil {
			bodyReader = bytes.NewReader(body)
		}
		req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), bodyReader)
		if err != nil {
			return nil, fmt.Errorf("create upstream request: %w", err)
		}

		for name, values := range headers {
			if strings.EqualFold(name, "host") {
				continue
			}
			for _, value := range values {
				req.Header.Add(name, value)
			}
		}

		token, err := c.tokenSource.GetToken(ctx, GetTokenOptions{App: c.app})
		if err != nil {
			return nil, fmt.Errorf("obtain token: %w", err)
		}
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := c.client.Do(req)
		if err != nil {
			return nil, fmt.Errorf("perform upstream request: %w", err)
		}

		if resp.StatusCode == http.StatusUnauthorized && attempt == 1 {
			resp.Body.Close()
			if _, err := c.tokenSource.GetToken(ctx, GetTokenOptions{App: c.app, Force: true}); err != nil {
				return nil, fmt.Errorf("refresh token: %w", err)
			}
			continue
		}

		return resp, nil
	}
}

// Manager provides lookup for clients based on endpoint names.
type Manager struct {
	clients map[string]*APIClient
}

// NewManager constructs API clients for all configured upstream services.
func NewManager(cfg config.Config, tokenMgr *TokenManager) (*Manager, error) {
	clients := make(map[string]*APIClient)

	create := func(name, app, base string) error {
		if strings.TrimSpace(base) == "" {
			return nil
		}
		client, err := NewAPIClient(app, base, tokenMgr)
		if err != nil {
			return err
		}
		clients[name] = client
		return nil
	}

	targets := []struct {
		name string
		app  string
		base string
	}{
		{"mophic", "mophic", cfg.MophICAPI},
		{"epidem", "mophic", cfg.EpidemAPI},
		{"phr", "mophic", cfg.MophPhrAPI},
		{"claim", "mophic", cfg.MophClaimAPI},
		{"fdh", "fdh", cfg.FDHAPI},
	}

	for _, target := range targets {
		if err := create(target.name, target.app, target.base); err != nil {
			return nil, fmt.Errorf("create client for %s: %w", target.name, err)
		}
	}

	return &Manager{clients: clients}, nil
}

// Client returns the API client mapped to the endpoint name.
func (m *Manager) Client(endpoint string) (*APIClient, error) {
	if endpoint == "" {
		endpoint = "mophic"
	}
	client, ok := m.clients[endpoint]
	if !ok {
		return nil, fmt.Errorf("unsupported endpoint %q", endpoint)
	}
	return client, nil
}
