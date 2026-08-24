# Coinbase Exchange SDK for Go

Go client for **Coinbase Exchange API** public market data. This module does not use Coinbase Advanced Trade API and does not implement authentication, accounts, orders, the user channel, Direct Market Data, FIX, persistence, redistribution, or K4K3RU integration.

Module path: `github.com/k4k3ru-hub/coinbase/go`

## Endpoints

| Environment | REST | Public WebSocket |
|---|---|---|
| Production | `https://api.exchange.coinbase.com` | `wss://ws-feed.exchange.coinbase.com` |
| Public sandbox | `https://api-public.sandbox.exchange.coinbase.com` | `wss://ws-feed-public.sandbox.exchange.coinbase.com` |

The sandbox URLs were checked against the official [Exchange Sandbox](https://docs.cdp.coinbase.com/exchange/introduction/sandbox) documentation on 2026-08-19.

## REST example

```go
client, err := rest.NewClient(nil)
if err != nil { return err }
book, err := client.MarketData().GetProductBook(ctx, "BTC-USD", marketdata.BookLevel2)
```

Implemented public operations: `/time`, `/currencies`, `/products`, `/products/{product_id}`, product `/book`, `/ticker`, `/trades`, `/candles`, `/stats`, and `/products/volume-summary`.

Book levels are deliberately different types. L1 and L2 rows contain price, size, and order count; L3 rows contain price, size, and order ID. Price, size, volume, and increment values remain decimal strings. IDs whose documented width may evolve are retained as JSON numbers rather than converted through `float64`.

Trades preserve `CB-BEFORE` and `CB-AFTER` cursor headers. The `side` of a trade or match is Coinbase Exchange's maker-order side; this SDK does not invert it into a taker-side interpretation.

Historic rates accept only official granularities `60`, `300`, `900`, `3600`, `21600`, and `86400` seconds and reject ranges exceeding 300 requested buckets. Exchange can return candles preceding `start`, and publishes no candle for intervals without ticks. The SDK neither fills nor interpolates gaps.

The ticker endpoint is a snapshot. Do not poll it at high frequency; use the WebSocket feed for real-time data. Historic rates also should not be polled frequently.

## WebSocket example

```go
client, err := websocket.NewClient(nil)
if err != nil { return err }
if err := client.Connect(ctx); err != nil { return err }
defer client.Close()
err = client.Subscribe(ctx, []websocket.Channel{
    {Name: websocket.ChannelHeartbeat},
    {Name: websocket.ChannelTicker},
    {Name: websocket.ChannelLevel2},
}, []string{"BTC-USD"})
```

Decoded channels and messages include `heartbeat`, `status`, `ticker`, `ticker_batch`, `level2`, `level2_batch`, `matches`/`last_match`, `subscriptions`, `error`, and documented Full object events (`received`, `open`, `done`, `match`, `change`, `activate`). Ticker batch uses the ticker schema but changes arrival behavior; level2 batch uses the L2 schema and is delivered every 50 ms.

For Coinbase Exchange, the official public-trade-only channel remains `matches`. `market_trades` belongs to Advanced Trade and is intentionally not exposed here. Unknown object message types are returned with raw JSON so additions do not break the entire stream.

Full support is raw typed event decoding only. This release does not provide an L3 builder or REST-L3/WebSocket synchronization. Consumers needing L3 must follow Coinbase's queue → REST level-3 snapshot → discard queued sequence `<= snapshot.sequence` → replay procedure with a bounded queue, invalidate on every sequence gap, and resynchronize after reconnect.

### L2 recovery and lifecycle

`BookManager` builds state only after a `snapshot`, applies `l2update` quantities as absolute quantities, and deletes a price when quantity is zero. `BestBidOffer` returns the maintained best prices and quantities without copying the full book. Reconnect, disconnect, or bounded consumer-queue overflow resets every book; updates are rejected until a new snapshot arrives. `level2` has no sequence field in the documented schema and Coinbase describes it as guaranteeing delivery, so the SDK does not invent a sequence check. Heartbeat sequence numbers can identify feed/trade gaps but do not prove that an L2 book is complete.

Exchange notes that messages on sequenced channels may be dropped or arrive out of order even though transport is TCP. This client preserves sequence fields and provides `SequenceTracker` to classify first, next, duplicate, gap, and out-of-order values independently per product. A tracker must be reset after reconnect. Because sequence semantics depend on the subscribed channel set, callers explicitly choose which compatible events to feed into it. The client does not automatically reconnect; the caller owns a bounded reconnect policy, observes terminal session errors through `SessionEnds`, and should restore subscriptions and wait for new L2 snapshots. Context cancellation closes the session, `Close` is idempotent, writes are serialized, event delivery is bounded, and no library logging occurs.

REST public endpoints are currently documented at 10 requests/second/IP with bursts up to 15; WebSocket limits are separate. The client does not automatically retry or rate-limit. A `429` preserves status, `Retry-After`, request ID when present, and cursor headers. Callers implementing retry must limit it to idempotent GETs, honor context and `Retry-After`, and use bounded backoff. See [REST rate limits](https://docs.cdp.coinbase.com/exchange/rest-api/rate-limits) and [WebSocket rate limits](https://docs.cdp.coinbase.com/exchange/websocket-feed/rate-limits).

## Market Data Terms

Access to a public endpoint is not permission for unlimited storage, caching, display, redistribution, derived-data use, or commercial use. The [Coinbase Market Data Terms of Use](https://www.coinbase.com/legal/market_data) were reviewed on 2026-08-19 (the page states last updated 2023-02-06):

- The stated license is limited, revocable, non-transferable, and non-sublicensable, for personal/research use by the individual or the entity's officers/employees.
- Third-party redistribution, display, or dissemination of Market Data and Derived Works requires prior express written consent.
- Certain derived works—including indexes, benchmarks, generic/fair-value prices, and financial-product valuations—are restricted even for internal use absent consent.
- The terms do not provide a general storage/caching grant. Review any retained copy against the terms and other applicable Coinbase agreements before implementation.
- Commercial or end-user applications are not authorized merely because the feed is public; obtain written permission or a suitable agreement.
- Coinbase retains ownership and trademark rights. The reviewed terms do not state a general attribution formula that substitutes for permission.
- Rate limits remain technical constraints in addition to license restrictions.
- Use must comply with applicable law. The terms select California law, subject to their federal-law qualification; jurisdiction-specific review may still be required.

This is an engineering summary, not legal advice. Confirm current terms, data package, attribution requirements, jurisdiction, and written permissions with Coinbase before storage, caching, external display, redistribution, derived-data publication, or commercial use.

## Testing

Unit tests use `httptest.Server` and injected fake WebSocket connections. They do not contact Coinbase or run live integration tests during normal `go test ./...` execution.
