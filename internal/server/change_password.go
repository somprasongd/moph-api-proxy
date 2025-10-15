package server

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"moph-api-proxy/internal/httpclient"
)

// ChangePasswordHandler handles credential rotation requests without proxying upstream.
type ChangePasswordHandler struct {
	tokens *httpclient.TokenManager
	logger *log.Logger
}

// NewChangePasswordHandler constructs a change password handler.
func NewChangePasswordHandler(tokens *httpclient.TokenManager, logger *log.Logger) *ChangePasswordHandler {
	return &ChangePasswordHandler{tokens: tokens, logger: logger}
}

// ServeHTTP implements http.Handler.
func (h *ChangePasswordHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	var payload struct {
		Username string `json:"username"`
		Password string `json:"password"`
		App      string `json:"app"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		h.logger.Printf("ERROR decode change-password payload: %v", err)
		writeJSONError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	payload.Username = strings.TrimSpace(payload.Username)
	payload.Password = strings.TrimSpace(payload.Password)
	payload.App = strings.TrimSpace(payload.App)

	if payload.Username == "" || payload.Password == "" {
		writeJSONError(w, http.StatusBadRequest, "username and password are required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	app := appOrDefault(payload.App)

	if _, err := h.tokens.GetToken(ctx, httpclient.GetTokenOptions{Force: true, Username: payload.Username, Password: payload.Password, App: app}); err != nil {
		h.logger.Printf("ERROR change-password token refresh failed: %v", err)
		writeJSONTokenError(w, err)
		return
	}

	h.logger.Printf("INFO change-password token refreshed app=%s username=%s", app, payload.Username)
	w.WriteHeader(http.StatusNoContent)
}

func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func writeJSONTokenError(w http.ResponseWriter, err error) {
	status := http.StatusBadRequest
	message := "failed to refresh token"

	if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusGatewayTimeout
		message = "token request timed out"
	} else if strings.Contains(err.Error(), "token endpoint returned 401") {
		status = http.StatusUnauthorized
		message = "invalid username or password"
	} else if strings.Contains(err.Error(), "token endpoint returned") {
		status = http.StatusBadGateway
	}

	writeJSONError(w, status, message)
}

func appOrDefault(app string) string {
	if strings.TrimSpace(app) == "" {
		return "mophic"
	}
	return app
}
