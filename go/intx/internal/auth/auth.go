package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const websocketDomain = "CBINTLMD"

// Credentials contains Coinbase International Exchange API credentials.
type Credentials struct {
	Key        string
	Secret     string
	Passphrase string
}

// Validate validates required credential fields.
//
// Version:
//   - 2026-08-24: Added.
func (c Credentials) Validate() error {
	if c.Key == "" {
		return fmt.Errorf("failed to validate coinbase intx credentials: key=empty")
	}
	if c.Secret == "" {
		return fmt.Errorf("failed to validate coinbase intx credentials: secret=empty")
	}
	if c.Passphrase == "" {
		return fmt.Errorf("failed to validate coinbase intx credentials: passphrase=empty")
	}
	return nil
}

// SignWebSocket signs the first INTX WebSocket subscription.
//
// Version:
//   - 2026-08-24: Added.
func SignWebSocket(c Credentials, timestamp string) (string, error) {
	if err := c.Validate(); err != nil {
		return "", fmt.Errorf("failed to sign coinbase intx websocket subscription: %w", err)
	}
	if timestamp == "" {
		return "", fmt.Errorf("failed to sign coinbase intx websocket subscription: timestamp=empty")
	}
	key, err := base64.StdEncoding.DecodeString(c.Secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign coinbase intx websocket subscription: failed to decode secret: %w", err)
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(timestamp + c.Key + websocketDomain + c.Passphrase))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}

// SignREST signs an INTX REST request.
//
// Version:
//   - 2026-08-24: Added.
func SignREST(c Credentials, timestamp, method, requestPath, body string) (string, error) {
	if err := c.Validate(); err != nil {
		return "", fmt.Errorf("failed to sign coinbase intx rest request: %w", err)
	}
	if timestamp == "" {
		return "", fmt.Errorf("failed to sign coinbase intx rest request: timestamp=empty")
	}
	if method == "" {
		return "", fmt.Errorf("failed to sign coinbase intx rest request: method=empty")
	}
	if requestPath == "" {
		return "", fmt.Errorf("failed to sign coinbase intx rest request: request_path=empty")
	}
	key, err := base64.StdEncoding.DecodeString(c.Secret)
	if err != nil {
		return "", fmt.Errorf("failed to sign coinbase intx rest request: failed to decode secret: %w", err)
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte(timestamp + method + requestPath + body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
}
