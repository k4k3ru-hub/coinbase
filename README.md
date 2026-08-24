# Coinbase Exchange SDK

A Go SDK for Coinbase Exchange public REST and WebSocket market data.

The Go module is located in [`go/`](go/README.md). Consumers import the
`rest`, `websocket`, and `marketdata` packages they require directly.

This SDK targets Coinbase Exchange, not Coinbase Advanced Trade. It does not
implement authentication, account access, order entry, persistence,
redistribution, or K4K3RU integration.
