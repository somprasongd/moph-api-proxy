package authpayload

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"moph-api-proxy/internal/cache"
	"moph-api-proxy/internal/config"
)

// Payload represents the authentication payload sent to the upstream service.
type Payload struct {
	User         string `json:"user"`
	PasswordHash string `json:"password_hash"`
	HospitalCode string `json:"hospital_code"`
}

// Create builds an authentication payload for the provided credentials.
func Create(username, password, secretKey, hospitalCode string) Payload {
	return Payload{
		User:         username,
		PasswordHash: hashPassword(password, secretKey),
		HospitalCode: hospitalCode,
	}
}

// IsCurrent checks whether the cached payload matches the provided credentials.
func IsCurrent(ctx context.Context, cacheClient *cache.Client, cfg config.Config, app, username, password string) (bool, error) {
	key := app + cfg.AuthPayloadKey
	value, err := cacheClient.Get(ctx, key)
	if err != nil || value == "" {
		return false, err
	}

	secret := cfg.MophICAuthSecret
	if app == "fdh" {
		secret = cfg.FDHAuthSecret
	}

	expected := Create(username, password, secret, cfg.MophHCode)
	buf, err := json.Marshal(expected)
	if err != nil {
		return false, err
	}

	return string(buf) == value, nil
}

func hashPassword(password, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(password))
	return hex.EncodeToString(h.Sum(nil))
}
