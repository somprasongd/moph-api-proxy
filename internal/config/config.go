package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config contains all runtime configuration values.
type Config struct {
	AppPort          int
	RedisHost        string
	RedisPort        int
	RedisPassword    string
	MophClaimAPI     string
	MophPhrAPI       string
	EpidemAPI        string
	FDHAPI           string
	FDHAuth          string
	FDHAuthSecret    string
	MophICAPI        string
	MophICAuth       string
	MophICAuthSecret string
	MophHCode        string
	UseAPIKey        bool
	TokenKeySuffix   string
	AuthPayloadKey   string
}

func envOrDefault(key, def string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return def
}

// Load creates a Config instance populated from environment variables.
func Load() (Config, error) {
	var cfg Config

	cfg.AppPort = parseInt(envOrDefault("APP_PORT", "3000"), 3000)
	cfg.RedisHost = envOrDefault("REDIS_HOST", "localhost")
	cfg.RedisPort = parseInt(envOrDefault("REDIS_PORT", "6379"), 6379)
	cfg.RedisPassword = os.Getenv("REDIS_PASSWORD")
	cfg.MophClaimAPI = envOrDefault("MOPH_CLAIM_API", "https://claim-nhso.moph.go.th")
	cfg.MophPhrAPI = envOrDefault("MOPH_PHR_API", "https://phr1.moph.go.th")
	cfg.EpidemAPI = envOrDefault("EPIDEM_API", "https://epidemcenter.moph.go.th/epidem")
	cfg.FDHAPI = envOrDefault("FDH_API", "https://fdh.moph.go.th")
	cfg.FDHAuth = envOrDefault("FDH_AUTH", "https://fdh.moph.go.th")
	cfg.FDHAuthSecret = envOrDefault("FDH_AUTH_SECRET", "$jwt@moph#")
	cfg.MophICAPI = envOrDefault("MOPH_IC_API", "https://cvp1.moph.go.th")
	cfg.MophICAuth = envOrDefault("MOPH_IC_AUTH", "https://cvp1.moph.go.th")
	cfg.MophICAuthSecret = envOrDefault("MOPH_IC_AUTH_SECRET", "$jwt@moph#")
	cfg.MophHCode = strings.TrimSpace(os.Getenv("MOPH_HCODE"))
	cfg.UseAPIKey = parseBool(envOrDefault("USE_API_KEY", "true"))
	cfg.TokenKeySuffix = envOrDefault("TOKEN_KEY", "-auth-token")
	cfg.AuthPayloadKey = envOrDefault("AUTH_PAYLOAD_KEY", "-auth-payload")

	if cfg.MophHCode == "" {
		return Config{}, errors.New("MOPH_HCODE is required")
	}

	return cfg, nil
}

func parseInt(value string, fallback int) int {
	if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		return v
	}
	return fallback
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "true", "1", "yes", "y":
		return true
	case "false", "0", "no", "n":
		return false
	default:
		// fallback to default true because original behaviour enabled API key.
		return true
	}
}

// Addr returns the listening address for the HTTP server.
func (c Config) Addr() string {
	return fmt.Sprintf(":%d", c.AppPort)
}
