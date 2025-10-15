package keygen

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Manager handles API key generation and validation.
type Manager struct {
	secret   string
	apiKey   string
	filePath string
}

// NewManager returns a new key manager pointing to the default key file location.
func NewManager() *Manager {
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	return &Manager{
		filePath: filepath.Join(wd, ".authorized_key", ".access.key"),
	}
}

// Init loads or generates the API secret and derived key.
func (m *Manager) Init() error {
	if m == nil {
		return errors.New("key manager is nil")
	}

	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create key directory: %w", err)
	}

	secretBytes, err := os.ReadFile(m.filePath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("read key file: %w", err)
		}
		secretBytes, err = generateSecret()
		if err != nil {
			return fmt.Errorf("generate secret: %w", err)
		}
		if err := os.WriteFile(m.filePath, secretBytes, 0o600); err != nil {
			return fmt.Errorf("write key file: %w", err)
		}
	}

	m.secret = strings.TrimSpace(string(secretBytes))
	m.apiKey = deriveAPIKey(m.secret)
	return nil
}

func generateSecret() ([]byte, error) {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return nil, err
	}
	buf := make([]byte, hex.EncodedLen(len(raw)))
	hex.Encode(buf, raw)
	return buf, nil
}

func deriveAPIKey(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	encoder := base32.StdEncoding.WithPadding(base32.NoPadding)
	return encoder.EncodeToString(hash[:])
}

// Verify ensures the provided API key matches the stored secret.
func (m *Manager) Verify(apiKey string) bool {
	if m == nil {
		return false
	}
	return apiKey != "" && strings.EqualFold(strings.TrimSpace(apiKey), m.apiKey)
}

// APIKey returns the current API key.
func (m *Manager) APIKey() string {
	if m == nil {
		return ""
	}
	return m.apiKey
}

// Secret returns the stored secret for testing/debugging purpose.
func (m *Manager) Secret() string {
	if m == nil {
		return ""
	}
	return m.secret
}
