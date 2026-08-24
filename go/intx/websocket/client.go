package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	gorilla "github.com/gorilla/websocket"
	"github.com/k4k3ru-hub/coinbase/go/intx/internal/auth"
)

const defaultQueueSize = 256

// Connection abstracts the underlying WebSocket connection.
type Connection interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	WriteControl(int, []byte, time.Time) error
	SetReadDeadline(time.Time) error
	SetPongHandler(func(string) error)
	Close() error
}

// Dialer creates WebSocket connections.
type Dialer interface {
	DialContext(context.Context, string, http.Header) (Connection, *http.Response, error)
}
type gorillaDialer struct{ d *gorilla.Dialer }

func (d gorillaDialer) DialContext(ctx context.Context, url string, header http.Header) (Connection, *http.Response, error) {
	return d.d.DialContext(ctx, url, header)
}

// Clock supplies authentication timestamps.
type Clock interface{ Now() time.Time }
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// ClientOption configures an INTX WebSocket client.
type ClientOption struct {
	EndpointURL  string
	Credentials  Credentials
	Dialer       Dialer
	Clock        Clock
	QueueSize    int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PingPeriod   time.Duration
}

// Credentials contains INTX WebSocket API credentials.
type Credentials struct {
	Key        string
	Secret     string
	Passphrase string
}

func (c Credentials) internal() auth.Credentials {
	return auth.Credentials{Key: c.Key, Secret: c.Secret, Passphrase: c.Passphrase}
}

// Client manages one authenticated INTX market-data session.
type Client struct {
	option  ClientOption
	mu      sync.Mutex
	writeMu sync.Mutex
	conn    Connection
	events  chan Event
	errs    chan error
	ends    chan error
	cancel  context.CancelFunc
	wg      sync.WaitGroup
	closed  bool
}

// DefaultClientOption returns production WebSocket defaults.
//
// Version:
//   - 2026-08-24: Added.
func DefaultClientOption(credentials Credentials) *ClientOption {
	return &ClientOption{EndpointURL: ProductionURL, Credentials: credentials, Dialer: gorillaDialer{d: gorilla.DefaultDialer}, Clock: systemClock{}, QueueSize: defaultQueueSize, ReadTimeout: 30 * time.Second, WriteTimeout: 5 * time.Second, PingPeriod: 15 * time.Second}
}

// NewClient creates a disconnected INTX WebSocket client.
//
// Version:
//   - 2026-08-24: Added.
func NewClient(option *ClientOption) (*Client, error) {
	if option == nil {
		return nil, fmt.Errorf("failed to create coinbase intx websocket client: option=null")
	}
	o := *option
	if o.EndpointURL == "" {
		o.EndpointURL = ProductionURL
	}
	if err := o.Credentials.internal().Validate(); err != nil {
		return nil, fmt.Errorf("failed to create coinbase intx websocket client: %w", err)
	}
	if o.Dialer == nil {
		return nil, fmt.Errorf("failed to create coinbase intx websocket client: dialer=null")
	}
	if o.Clock == nil {
		return nil, fmt.Errorf("failed to create coinbase intx websocket client: clock=null")
	}
	if o.QueueSize == 0 {
		o.QueueSize = defaultQueueSize
	}
	if o.QueueSize < 0 {
		return nil, fmt.Errorf("failed to create coinbase intx websocket client: queue_size=out_of_range")
	}
	return &Client{option: o, events: make(chan Event, o.QueueSize), errs: make(chan error, 1), ends: make(chan error, 1)}, nil
}

// Connect opens a fresh INTX WebSocket session.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("failed to connect coinbase intx websocket: client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to connect coinbase intx websocket: context=null")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("failed to connect coinbase intx websocket: client=closed")
	}
	if c.conn != nil {
		return fmt.Errorf("failed to connect coinbase intx websocket: connection=already_created")
	}
	conn, _, err := c.option.Dialer.DialContext(ctx, c.option.EndpointURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect coinbase intx websocket: %w", err)
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	c.conn = conn
	c.cancel = cancel
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(c.option.Clock.Now().Add(c.option.ReadTimeout)) })
	c.wg.Add(3)
	go c.readLoop(sessionCtx, conn)
	go c.pingLoop(sessionCtx, conn)
	go c.contextLoop(sessionCtx, conn)
	return nil
}

// Subscribe authenticates and subscribes to perpetual-only channels.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Subscribe(ctx context.Context, channels, productIDs []string) error {
	return c.send(ctx, "SUBSCRIBE", channels, productIDs, true)
}

// Unsubscribe removes perpetual-only subscriptions.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Unsubscribe(ctx context.Context, channels, productIDs []string) error {
	return c.send(ctx, "UNSUBSCRIBE", channels, productIDs, false)
}

func (c *Client) send(ctx context.Context, typ string, channels, productIDs []string, authenticate bool) error {
	if c == nil {
		return fmt.Errorf("failed to %s coinbase intx websocket: client=null", strings.ToLower(typ))
	}
	if ctx == nil {
		return fmt.Errorf("failed to %s coinbase intx websocket: context=null", strings.ToLower(typ))
	}
	if len(channels) == 0 {
		return fmt.Errorf("failed to %s coinbase intx websocket: channels=empty", strings.ToLower(typ))
	}
	if len(productIDs) == 0 {
		return fmt.Errorf("failed to %s coinbase intx websocket: product_ids=empty", strings.ToLower(typ))
	}
	for _, id := range productIDs {
		if !strings.HasSuffix(id, "-PERP") {
			return fmt.Errorf("failed to %s coinbase intx websocket: product_id=invalid", strings.ToLower(typ))
		}
	}
	req := SubscriptionRequest{Type: typ, Channels: channels, ProductIDs: productIDs}
	if authenticate {
		ts := fmt.Sprintf("%d", c.option.Clock.Now().Unix())
		signature, err := auth.SignWebSocket(c.option.Credentials.internal(), ts)
		if err != nil {
			return fmt.Errorf("failed to subscribe coinbase intx websocket: %w", err)
		}
		req.Time = ts
		req.Key = c.option.Credentials.Key
		req.Passphrase = c.option.Credentials.Passphrase
		req.Signature = signature
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to %s coinbase intx websocket: %w", strings.ToLower(typ), err)
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("failed to %s coinbase intx websocket: connection=null", strings.ToLower(typ))
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := conn.WriteMessage(gorilla.TextMessage, payload); err != nil {
		return fmt.Errorf("failed to %s coinbase intx websocket: %w", strings.ToLower(typ), err)
	}
	return nil
}

// Events returns the bounded decoded event stream.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Events() <-chan Event {
	if c == nil {
		return nil
	}
	return c.events
}

// Errors returns non-terminal asynchronous errors.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Errors() <-chan error {
	if c == nil {
		return nil
	}
	return c.errs
}

// SessionEnds returns terminal session errors.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) SessionEnds() <-chan error {
	if c == nil {
		return nil
	}
	return c.ends
}

// Close idempotently closes the client.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Close() error {
	if c == nil {
		return nil
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	cancel := c.cancel
	conn := c.conn
	c.conn = nil
	c.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	var err error
	if conn != nil {
		err = conn.Close()
	}
	c.wg.Wait()
	return err
}

func (c *Client) readLoop(ctx context.Context, conn Connection) {
	defer c.wg.Done()
	for {
		if c.option.ReadTimeout > 0 {
			_ = conn.SetReadDeadline(c.option.Clock.Now().Add(c.option.ReadTimeout))
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			c.sessionEnded(conn, fmt.Errorf("failed to read coinbase intx websocket: %w", err))
			return
		}
		event, err := DecodeEvent(data)
		if err != nil {
			c.report(err)
			continue
		}
		select {
		case c.events <- event:
		case <-ctx.Done():
			return
		default:
			c.report(fmt.Errorf("failed to route coinbase intx websocket event: consumer=too_slow queue_size=%d", cap(c.events)))
		}
	}
}
func (c *Client) pingLoop(ctx context.Context, conn Connection) {
	defer c.wg.Done()
	if c.option.PingPeriod <= 0 {
		return
	}
	ticker := time.NewTicker(c.option.PingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.writeMu.Lock()
			err := conn.WriteControl(gorilla.PingMessage, nil, c.option.Clock.Now().Add(c.option.WriteTimeout))
			c.writeMu.Unlock()
			if err != nil {
				c.sessionEnded(conn, fmt.Errorf("failed to ping coinbase intx websocket: %w", err))
				return
			}
		}
	}
}
func (c *Client) contextLoop(ctx context.Context, conn Connection) {
	defer c.wg.Done()
	<-ctx.Done()
	c.mu.Lock()
	if c.conn == conn {
		c.conn = nil
		_ = conn.Close()
	}
	c.mu.Unlock()
}
func (c *Client) sessionEnded(conn Connection, err error) {
	c.mu.Lock()
	if c.conn != conn {
		c.mu.Unlock()
		return
	}
	c.conn = nil
	if c.cancel != nil {
		c.cancel()
	}
	c.mu.Unlock()
	_ = conn.Close()
	select {
	case c.ends <- err:
	default:
	}
}
func (c *Client) report(err error) {
	select {
	case c.errs <- err:
	default:
	}
}
