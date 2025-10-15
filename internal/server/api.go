package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"moph-ic-proxy/internal/httpclient"
)

// ProxyHandler forwards API requests to configured upstream endpoints.
type ProxyHandler struct {
	clients *httpclient.Manager
	logger  *log.Logger
}

// NewProxyHandler constructs a new proxy handler.
func NewProxyHandler(clients *httpclient.Manager, logger *log.Logger) *ProxyHandler {
	return &ProxyHandler{clients: clients, logger: logger}
}

// ServeHTTP implements http.Handler.
func (h *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	endpoint := r.Header.Get("X-API-ENDPOINT")
	query := r.URL.Query()
	if endpoint == "" {
		endpoint = query.Get("endpoint")
	}
	if endpoint != "" {
		queryCopy := cloneValues(query)
		queryCopy.Del("endpoint")
		r.URL.RawQuery = queryCopy.Encode()
	}

	client, err := h.clients.Client(endpoint)
	if err != nil {
		h.logger.Printf("ERROR resolve client: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	path := r.URL.Path
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	var body []byte
	if r.Method != http.MethodGet && r.ContentLength != 0 {
		buf, err := io.ReadAll(r.Body)
		if err != nil {
			h.logger.Printf("ERROR read request body: %v", err)
			http.Error(w, "failed to read request body", http.StatusBadRequest)
			return
		}
		body = buf
	}

	h.logger.Printf("INFO proxy %s %s endpoint=%s", r.Method, r.URL.String(), endpoint)
	resp, err := client.Do(ctx, r.Method, path, r.URL.RawQuery, body, r.Header)
	if err != nil {
		h.logger.Printf("ERROR proxy request failed: %v", err)
		http.Error(w, fmt.Sprintf("proxy error: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	if _, err := io.Copy(w, resp.Body); err != nil {
		h.logger.Printf("ERROR writing response body: %v", err)
	}
}

func cloneValues(v url.Values) url.Values {
	copy := make(url.Values, len(v))
	for key, values := range v {
		copy[key] = append([]string(nil), values...)
	}
	return copy
}
