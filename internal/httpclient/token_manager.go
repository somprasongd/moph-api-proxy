package httpclient

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"moph-ic-proxy/internal/authpayload"
	"moph-ic-proxy/internal/cache"
	"moph-ic-proxy/internal/config"
)

// TokenManager provides caching and refresh logic for application tokens.
type TokenManager struct {
	cfg        config.Config
	cache      *cache.Client
	httpClient *http.Client
}

// GetTokenOptions controls how tokens are requested.
type GetTokenOptions struct {
	Force    bool
	Username string
	Password string
	App      string
}

// NewTokenManager creates a token manager with a dedicated HTTP client.
func NewTokenManager(cfg config.Config, cacheClient *cache.Client) *TokenManager {
	return &TokenManager{
		cfg:   cfg,
		cache: cacheClient,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				Proxy:               http.ProxyFromEnvironment,
				MaxIdleConns:        100,
				IdleConnTimeout:     90 * time.Second,
				MaxIdleConnsPerHost: 10,
			},
		},
	}
}

// GetToken obtains an access token for the requested application.
func (m *TokenManager) GetToken(ctx context.Context, opts GetTokenOptions) (string, error) {
	if opts.App == "" {
		opts.App = "mophic"
	}

	tokenKey := opts.App + m.cfg.TokenKeySuffix
	payloadKey := opts.App + m.cfg.AuthPayloadKey

	if opts.Force {
		if err := m.cache.Del(ctx, tokenKey); err != nil {
			return "", fmt.Errorf("delete token cache: %w", err)
		}
	}

	if !opts.Force {
		cachedToken, err := m.cache.Get(ctx, tokenKey)
		if err != nil {
			return "", fmt.Errorf("read token cache: %w", err)
		}
		if strings.TrimSpace(cachedToken) != "" {
			return cachedToken, nil
		}
	}

	var payload authpayload.Payload

	secret := m.cfg.MophICAuthSecret
	baseURL := m.cfg.MophICAuth
	if opts.App == "fdh" {
		secret = m.cfg.FDHAuthSecret
		baseURL = m.cfg.FDHAuth
	}

	if opts.Username != "" && opts.Password != "" {
		payload = authpayload.Create(opts.Username, opts.Password, secret, m.cfg.MophHCode)
	} else {
		saved, err := m.cache.Get(ctx, payloadKey)
		if err != nil {
			return "", fmt.Errorf("read payload cache: %w", err)
		}
		if saved == "" {
			return "", errors.New("no cached credentials available")
		}
		if err := json.Unmarshal([]byte(saved), &payload); err != nil {
			return "", fmt.Errorf("decode cached payload: %w", err)
		}
	}

	buf, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshal payload: %w", err)
	}

	token, err := m.requestToken(ctx, baseURL, buf)
	if err != nil {
		return "", err
	}

	if err := m.cache.Set(ctx, payloadKey, string(buf)); err != nil {
		return "", fmt.Errorf("store auth payload: %w", err)
	}

	expiresAt, err := extractExpiry(token)
	if err != nil {
		return "", err
	}

	expiresAt -= 60
	if expiresAt <= 0 {
		expiresAt = time.Now().Add(5 * time.Minute).Unix()
	}

	if err := m.cache.SetWithExpireAt(ctx, tokenKey, token, expiresAt); err != nil {
		return "", fmt.Errorf("store token: %w", err)
	}

	return token, nil
}

func (m *TokenManager) requestToken(ctx context.Context, baseURL string, payload []byte) (string, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("parse token base URL: %w", err)
	}
	u.Path = strings.TrimSuffix(u.Path, "/") + "/token"
	query := u.Query()
	query.Set("Action", "get_moph_access_token")
	u.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("perform token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	token := strings.TrimSpace(string(body))
	if token == "" {
		return "", errors.New("empty token received")
	}

	return token, nil
}

func extractExpiry(token string) (int64, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return 0, errors.New("invalid token format")
	}
	payloadSegment := parts[1]
	decoded, err := base64.RawURLEncoding.DecodeString(payloadSegment)
	if err != nil {
		return 0, fmt.Errorf("decode token payload: %w", err)
	}

	var claims map[string]any
	if err := json.Unmarshal(decoded, &claims); err != nil {
		return 0, fmt.Errorf("unmarshal token payload: %w", err)
	}

	switch exp := claims["exp"].(type) {
	case float64:
		return int64(exp), nil
	case json.Number:
		v, err := exp.Int64()
		if err != nil {
			return 0, err
		}
		return v, nil
	default:
		return 0, errors.New("token payload missing exp claim")
	}
}
