package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestSignWebSocket(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("secret"))
	credentials := Credentials{Key: "key", Secret: secret, Passphrase: "pass"}
	got, err := SignWebSocket(credentials, "100")
	if err != nil {
		t.Fatalf("SignWebSocket() error = %v", err)
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte("100keyCBINTLMDpass"))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("SignWebSocket() = %q, want %q", got, want)
	}
}

func TestSignREST(t *testing.T) {
	secret := base64.StdEncoding.EncodeToString([]byte("secret"))
	credentials := Credentials{Key: "key", Secret: secret, Passphrase: "pass"}
	got, err := SignREST(credentials, "100", "GET", "/api/v1/instruments", "")
	if err != nil {
		t.Fatalf("SignREST() error = %v", err)
	}
	mac := hmac.New(sha256.New, []byte("secret"))
	_, _ = mac.Write([]byte("100GET/api/v1/instruments"))
	want := base64.StdEncoding.EncodeToString(mac.Sum(nil))
	if got != want {
		t.Fatalf("SignREST() = %q, want %q", got, want)
	}
}
