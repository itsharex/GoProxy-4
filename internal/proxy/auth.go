package proxy

import (
	"strings"

	"gitee.com/jiuhuidalan1/goproxy/internal/config"
	"golang.org/x/crypto/bcrypt"
)

// Authenticator validates optional username/password authentication.
type Authenticator interface {
	Enabled() bool
	Validate(username, password string) bool
}

// AuthManager validates users against bcrypt password hashes.
type AuthManager struct {
	enabled bool
	users   map[string]string
}

// NewAuthManager creates an immutable authentication manager from config.
func NewAuthManager(cfg config.AuthConfig) *AuthManager {
	users := make(map[string]string, len(cfg.Users))
	for _, user := range cfg.Users {
		username := strings.TrimSpace(user.Username)
		if username == "" || user.Password == "" {
			continue
		}
		users[username] = user.Password
	}
	return &AuthManager{
		enabled: cfg.Enabled,
		users:   users,
	}
}

// HashPassword returns a bcrypt hash suitable for config.AuthUser.Password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Enabled reports whether authentication is required.
func (a *AuthManager) Enabled() bool {
	return a != nil && a.enabled
}

// Validate checks the supplied credentials.
func (a *AuthManager) Validate(username, password string) bool {
	if a == nil || !a.enabled {
		return true
	}
	hash, ok := a.users[username]
	if !ok {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
