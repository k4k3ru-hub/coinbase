// Package coinbaseexchange provides the root facade for Coinbase Exchange public market data.
package coinbaseexchange

import (
	"github.com/k4k3ru-hub/coinbase-exchange/go/rest"
	ws "github.com/k4k3ru-hub/coinbase-exchange/go/websocket"
)

type RESTClient = rest.Client
type RESTClientOption = rest.ClientOption
type WebSocketClient = ws.Client
type WebSocketClientOption = ws.ClientOption

// NewRESTClient creates the public Coinbase Exchange REST client.
//
// Version:
//   - 2026-08-19: Added.
func NewRESTClient(option *RESTClientOption) (*RESTClient, error) { return rest.NewClient(option) }

// NewWebSocketClient creates a disconnected public Coinbase Exchange WebSocket client.
//
// Version:
//   - 2026-08-19: Added.
func NewWebSocketClient(option *WebSocketClientOption) (*WebSocketClient, error) {
	return ws.NewClient(option)
}
