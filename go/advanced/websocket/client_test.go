package websocket

import (
	"context"
	"net/http"
	"testing"
)

type testDialer struct{}

func (testDialer) DialContext(context.Context, string, http.Header) (Connection, *http.Response, error) {
	return nil, nil, nil
}

func TestNewClientRequiresDialer(t *testing.T) {
	if _, err := NewClient(&ClientOption{}); err == nil {
		t.Fatal("NewClient() error = nil")
	}
}

func TestSubscribeRejectsNonPerpetualProduct(t *testing.T) {
	client, err := NewClient(&ClientOption{Dialer: testDialer{}})
	if err != nil {
		t.Fatal(err)
	}
	if err := client.Subscribe(context.Background(), ChannelLevel2, []string{"BTC-USD"}); err == nil {
		t.Fatal("Subscribe() error = nil")
	}
}

func TestSubscribeAllowsHeartbeatWithoutProducts(t *testing.T) {
	client, err := NewClient(&ClientOption{Dialer: testDialer{}})
	if err != nil {
		t.Fatal(err)
	}
	err = client.Subscribe(context.Background(), ChannelHeartbeats, nil)
	if err == nil {
		t.Fatal("Subscribe() error = nil without connection")
	}
	if got := err.Error(); got != "failed to subscribe coinbase advanced websocket: connection=null" {
		t.Fatalf("error = %q", got)
	}
}
