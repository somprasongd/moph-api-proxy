package middleware

import (
	"log"
	"net/http"
	"net/url"

	"moph-api-proxy/internal/config"
	"moph-api-proxy/internal/keygen"
)

// APIKeyVerifier ensures that incoming requests present a valid API key when enabled.
func APIKeyVerifier(cfg config.Config, manager *keygen.Manager, logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.UseAPIKey {
				next.ServeHTTP(w, r)
				return
			}

			key := r.Header.Get("X-API-KEY")
			if key == "" {
				key = r.URL.Query().Get("x-api-key")
			}
			if key == "" || !manager.Verify(key) {
				logger.Printf("WARN invalid or missing API key from %s", r.RemoteAddr)
				http.Error(w, "invalid or missing API key", http.StatusUnauthorized)
				return
			}

			if q := r.URL.Query(); q.Has("x-api-key") {
				cloned := cloneValues(q)
				cloned.Del("x-api-key")
				r.URL.RawQuery = cloned.Encode()
			}

			next.ServeHTTP(w, r)
		})
	}
}

func cloneValues(v url.Values) url.Values {
	copy := make(url.Values, len(v))
	for key, values := range v {
		copy[key] = append([]string(nil), values...)
	}
	return copy
}
