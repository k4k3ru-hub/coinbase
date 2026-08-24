package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	gorilla "github.com/gorilla/websocket"
)

const defaultQueueSize = 1024

// Connection abstracts the underlying WebSocket implementation.
type Connection interface {
	ReadMessage() (int, []byte, error)
	WriteMessage(int, []byte) error
	WriteControl(int, []byte, time.Time) error
	SetReadDeadline(time.Time) error
	SetPongHandler(func(string) error)
	Close() error
}

type writeDeadlineSetter interface {
	SetWriteDeadline(time.Time) error
}

// Dialer creates WebSocket connections.
type Dialer interface {
	DialContext(context.Context, string, http.Header) (Connection, *http.Response, error)
}
type gorillaDialer struct{ d *gorilla.Dialer }

func (d gorillaDialer) DialContext(ctx context.Context, u string, h http.Header) (Connection, *http.Response, error) {
	return d.d.DialContext(ctx, u, h)
}

// ClientOption configures a WebSocket client.
type ClientOption struct {
	EndpointURL                           string
	Dialer                                Dialer
	QueueSize                             int
	ReadTimeout, WriteTimeout, PingPeriod time.Duration
}

// Client manages one public Exchange WebSocket session.
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
	books   *BookManager
}

// DefaultClientOption returns public production WebSocket defaults.
//
// Version:
//   - 2026-08-25: Increased the default event queue capacity.
//   - 2026-08-19: Added.
func DefaultClientOption() *ClientOption {
	return &ClientOption{EndpointURL: ProductionURL, Dialer: gorillaDialer{d: gorilla.DefaultDialer}, QueueSize: defaultQueueSize, ReadTimeout: 30 * time.Second, WriteTimeout: 5 * time.Second, PingPeriod: 15 * time.Second}
}

// NewClient creates a disconnected WebSocket client.
//
// Version:
//   - 2026-08-19: Added.
func NewClient(option *ClientOption) (*Client, error) {
	if option == nil {
		option = DefaultClientOption()
	}
	o := *option
	if o.EndpointURL == "" {
		o.EndpointURL = ProductionURL
	}
	if o.Dialer == nil {
		return nil, fmt.Errorf("failed to create coinbase exchange websocket client: dialer=null")
	}
	if o.QueueSize == 0 {
		o.QueueSize = defaultQueueSize
	}
	if o.QueueSize < 0 {
		return nil, fmt.Errorf("failed to create coinbase exchange websocket client: queue_size=out_of_range")
	}
	c := &Client{option: o, events: make(chan Event, o.QueueSize), errs: make(chan error, 1), ends: make(chan error, 1)}
	c.books = NewBookManager()
	return c, nil
}

// Connect opens a session. Calling Connect after a disconnect creates a fresh book generation.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("failed to connect coinbase exchange websocket: client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to connect coinbase exchange websocket: context=null")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("failed to connect coinbase exchange websocket: client=closed")
	}
	if c.conn != nil {
		return fmt.Errorf("failed to connect coinbase exchange websocket: connection=already_created")
	}
	conn, _, err := c.option.Dialer.DialContext(ctx, c.option.EndpointURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect coinbase exchange websocket: %w", err)
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	c.conn = conn
	c.cancel = cancel
	c.books.Reset()
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(c.option.ReadTimeout)) })
	c.wg.Add(3)
	go c.readLoop(sessionCtx, conn)
	go c.pingLoop(sessionCtx, conn)
	go c.contextLoop(sessionCtx, conn)
	return nil
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

// Subscribe sends a serialized subscription request.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Subscribe(ctx context.Context, channels []Channel, productIDs []string) error {
	return c.send(ctx, "subscribe", channels, productIDs)
}

// Unsubscribe sends a serialized unsubscription request.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Unsubscribe(ctx context.Context, channels []Channel, productIDs []string) error {
	return c.send(ctx, "unsubscribe", channels, productIDs)
}
func (c *Client) send(ctx context.Context, typ string, channels []Channel, ids []string) error {
	if c == nil {
		return fmt.Errorf("failed to %s coinbase exchange websocket: client=null", typ)
	}
	if ctx == nil {
		return fmt.Errorf("failed to %s coinbase exchange websocket: context=null", typ)
	}
	if len(channels) == 0 {
		return fmt.Errorf("failed to %s coinbase exchange websocket: channels=empty", typ)
	}
	for _, ch := range channels {
		if ch.Name == "" {
			return fmt.Errorf("failed to %s coinbase exchange websocket: channel_name=empty", typ)
		}
	}
	payload, err := json.Marshal(SubscriptionRequest{Type: typ, Channels: channels, ProductIDs: ids})
	if err != nil {
		return fmt.Errorf("failed to %s coinbase exchange websocket: %w", typ, err)
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("failed to %s coinbase exchange websocket: connection=null", typ)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	deadline, _ := ctx.Deadline()
	if setter, ok := conn.(writeDeadlineSetter); ok {
		if err := setter.SetWriteDeadline(deadline); err != nil {
			return fmt.Errorf("failed to %s coinbase exchange websocket: failed to set write deadline: %w", typ, err)
		}
	}
	if err := conn.WriteMessage(gorilla.TextMessage, payload); err != nil {
		return fmt.Errorf("failed to %s coinbase exchange websocket: %w", typ, err)
	}
	return nil
}

// Events returns the bounded decoded event stream.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Events() <-chan Event {
	if c == nil {
		return nil
	}
	return c.events
}

// Errors returns asynchronous connection, decode, and slow-consumer errors.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Errors() <-chan error {
	if c == nil {
		return nil
	}
	return c.errs
}

// SessionEnds returns reliable terminal errors for disconnected sessions.
//
// Returns:
//   - Session termination error stream.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) SessionEnds() <-chan error {
	if c == nil {
		return nil
	}
	return c.ends
}

// Books returns the generation-aware L2 book manager.
//
// Version:
//   - 2026-08-19: Added.
func (c *Client) Books() *BookManager {
	if c == nil {
		return nil
	}
	return c.books
}

// Close idempotently stops the current session and client.
//
// Version:
//   - 2026-08-19: Added.
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
			_ = conn.SetReadDeadline(time.Now().Add(c.option.ReadTimeout))
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			c.sessionEnded(conn, fmt.Errorf("failed to read coinbase exchange websocket: %w", err))
			return
		}
		event, err := DecodeEvent(data)
		if err != nil {
			c.report(err)
			continue
		}
		if err := c.books.Apply(event); err != nil {
			c.report(err)
			continue
		}
		select {
		case c.events <- event:
		case <-ctx.Done():
			return
		default:
			c.sessionEnded(conn, fmt.Errorf("failed to route coinbase exchange websocket event: consumer=too_slow queue_size=%d", cap(c.events)))
			return
		}
	}
}
func (c *Client) pingLoop(ctx context.Context, conn Connection) {
	defer c.wg.Done()
	if c.option.PingPeriod <= 0 {
		return
	}
	t := time.NewTicker(c.option.PingPeriod)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			c.writeMu.Lock()
			err := conn.WriteControl(gorilla.PingMessage, nil, time.Now().Add(c.option.WriteTimeout))
			c.writeMu.Unlock()
			if err != nil {
				c.sessionEnded(conn, fmt.Errorf("failed to ping coinbase exchange websocket: %w", err))
				return
			}
		}
	}
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
	c.books.Reset()
	c.ends <- err
}
func (c *Client) report(err error) {
	select {
	case c.errs <- err:
	default:
	}
}
