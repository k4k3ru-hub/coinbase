package websocket

import (
	"context"
	"encoding/base64"
	"net/http"
	"testing"
	"time"
)

type testDialer struct{}

func (testDialer) DialContext(context.Context, string, http.Header) (Connection, *http.Response, error) {
	return nil, nil, nil
}

type testClock struct{}

func (testClock) Now() time.Time { return time.Unix(100, 0) }

func TestNewClientRequiresCredentials(t *testing.T) {
	_, err := NewClient(&ClientOption{Dialer: testDialer{}, Clock: testClock{}})
	if err == nil {
		t.Fatal("NewClient() error = nil")
	}
}

func TestSubscribeRejectsSpotProduct(t *testing.T) {
	client, err := NewClient(&ClientOption{Credentials: Credentials{Key: "key", Secret: base64.StdEncoding.EncodeToString([]byte("secret")), Passphrase: "pass"}, Dialer: testDialer{}, Clock: testClock{}})
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	err = client.Subscribe(context.Background(), []string{ChannelLevel1}, []string{"BTC-USDC"})
	if err == nil {
		t.Fatal("Subscribe() error = nil")
	}
}
