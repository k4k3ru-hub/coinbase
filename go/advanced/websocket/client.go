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

// ClientOption configures an anonymous Advanced Trade WebSocket client.
type ClientOption struct {
	EndpointURL  string
	Dialer       Dialer
	QueueSize    int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	PingPeriod   time.Duration
}

// Client manages one Advanced Trade public market-data session.
type Client struct {
	option        ClientOption
	mu            sync.Mutex
	writeMu       sync.Mutex
	conn          Connection
	events        chan Event
	errs          chan error
	ends          chan error
	cancel        context.CancelFunc
	wg            sync.WaitGroup
	closed        bool
	books         *BookManager
	sequences     map[string]uint64
	seenSequences map[string]bool
}

// DefaultClientOption returns anonymous production WebSocket defaults.
//
// Version:
//   - 2026-08-24: Added.
func DefaultClientOption() *ClientOption {
	return &ClientOption{EndpointURL: ProductionURL, Dialer: gorillaDialer{d: gorilla.DefaultDialer}, QueueSize: defaultQueueSize, ReadTimeout: 30 * time.Second, WriteTimeout: 5 * time.Second, PingPeriod: 15 * time.Second}
}

// NewClient creates a disconnected anonymous Advanced Trade WebSocket client.
//
// Version:
//   - 2026-08-24: Added.
func NewClient(option *ClientOption) (*Client, error) {
	if option == nil {
		option = DefaultClientOption()
	}
	o := *option
	if o.EndpointURL == "" {
		o.EndpointURL = ProductionURL
	}
	if o.Dialer == nil {
		return nil, fmt.Errorf("failed to create coinbase advanced websocket client: dialer=null")
	}
	if o.QueueSize == 0 {
		o.QueueSize = defaultQueueSize
	}
	if o.QueueSize < 0 {
		return nil, fmt.Errorf("failed to create coinbase advanced websocket client: queue_size=out_of_range")
	}
	return &Client{option: o, events: make(chan Event, o.QueueSize), errs: make(chan error, 1), ends: make(chan error, 1), books: NewBookManager(), sequences: map[string]uint64{}, seenSequences: map[string]bool{}}, nil
}

// Connect opens a fresh Advanced Trade public market-data session.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Connect(ctx context.Context) error {
	if c == nil {
		return fmt.Errorf("failed to connect coinbase advanced websocket: client=null")
	}
	if ctx == nil {
		return fmt.Errorf("failed to connect coinbase advanced websocket: context=null")
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return fmt.Errorf("failed to connect coinbase advanced websocket: client=closed")
	}
	if c.conn != nil {
		return fmt.Errorf("failed to connect coinbase advanced websocket: connection=already_created")
	}
	conn, _, err := c.option.Dialer.DialContext(ctx, c.option.EndpointURL, nil)
	if err != nil {
		return fmt.Errorf("failed to connect coinbase advanced websocket: %w", err)
	}
	sessionCtx, cancel := context.WithCancel(ctx)
	c.conn = conn
	c.cancel = cancel
	c.books.Reset()
	c.sequences = map[string]uint64{}
	c.seenSequences = map[string]bool{}
	conn.SetPongHandler(func(string) error { return conn.SetReadDeadline(time.Now().Add(c.option.ReadTimeout)) })
	c.wg.Add(3)
	go c.readLoop(sessionCtx, conn)
	go c.pingLoop(sessionCtx, conn)
	go c.contextLoop(sessionCtx, conn)
	return nil
}

// Subscribe subscribes anonymously to one public channel.
//
// Parameters:
//   - channel: public Advanced Trade channel.
//   - productIDs: perpetual product IDs; empty only for heartbeats or status.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Subscribe(ctx context.Context, channel string, productIDs []string) error {
	return c.send(ctx, "subscribe", channel, productIDs)
}

// Unsubscribe removes one public channel subscription.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Unsubscribe(ctx context.Context, channel string, productIDs []string) error {
	return c.send(ctx, "unsubscribe", channel, productIDs)
}

func (c *Client) send(ctx context.Context, typ, channel string, productIDs []string) error {
	if c == nil {
		return fmt.Errorf("failed to %s coinbase advanced websocket: client=null", typ)
	}
	if ctx == nil {
		return fmt.Errorf("failed to %s coinbase advanced websocket: context=null", typ)
	}
	if channel == "" {
		return fmt.Errorf("failed to %s coinbase advanced websocket: channel=empty", typ)
	}
	if len(productIDs) == 0 && channel != ChannelHeartbeats && channel != ChannelStatus {
		return fmt.Errorf("failed to %s coinbase advanced websocket: product_ids=empty", typ)
	}
	for _, productID := range productIDs {
		if !strings.HasSuffix(productID, "-PERP-INTX") {
			return fmt.Errorf("failed to %s coinbase advanced websocket: product_id=invalid", typ)
		}
	}
	payload, err := json.Marshal(SubscriptionRequest{Type: typ, Channel: channel, ProductIDs: productIDs})
	if err != nil {
		return fmt.Errorf("failed to %s coinbase advanced websocket: %w", typ, err)
	}
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return fmt.Errorf("failed to %s coinbase advanced websocket: connection=null", typ)
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if err := conn.WriteMessage(gorilla.TextMessage, payload); err != nil {
		return fmt.Errorf("failed to %s coinbase advanced websocket: %w", typ, err)
	}
	return nil
}

// Events returns the bounded decoded public event stream.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Events() <-chan Event {
	if c == nil {
		return nil
	}
	return c.events
}

// Errors returns non-terminal decode, sequence, book, and overflow errors.
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

// Books returns the generation-aware L2 book manager.
//
// Version:
//   - 2026-08-24: Added.
func (c *Client) Books() *BookManager {
	if c == nil {
		return nil
	}
	return c.books
}

// Close idempotently stops the current session and client.
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
			_ = conn.SetReadDeadline(time.Now().Add(c.option.ReadTimeout))
		}
		_, data, err := conn.ReadMessage()
		if err != nil {
			c.sessionEnded(conn, fmt.Errorf("failed to read coinbase advanced websocket: %w", err))
			return
		}
		event, err := DecodeEvent(data)
		if err != nil {
			c.report(err)
			continue
		}
		if c.seenSequences[event.Channel] && event.SequenceNum != c.sequences[event.Channel]+1 {
			if event.Channel == ChannelLevel2 || event.Channel == ChannelLevel2Data {
				c.books.Reset()
			}
			c.report(fmt.Errorf("failed to validate coinbase advanced websocket sequence: sequence=invalid channel=%q previous_sequence=%d sequence=%d", event.Channel, c.sequences[event.Channel], event.SequenceNum))
		}
		c.sequences[event.Channel] = event.SequenceNum
		c.seenSequences[event.Channel] = true
		if err := c.books.Apply(event); err != nil {
			c.report(err)
			continue
		}
		select {
		case c.events <- event:
		case <-ctx.Done():
			return
		default:
			c.books.Reset()
			c.report(fmt.Errorf("failed to route coinbase advanced websocket event: consumer=too_slow queue_size=%d", cap(c.events)))
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
			err := conn.WriteControl(gorilla.PingMessage, nil, time.Now().Add(c.option.WriteTimeout))
			c.writeMu.Unlock()
			if err != nil {
				c.sessionEnded(conn, fmt.Errorf("failed to ping coinbase advanced websocket: %w", err))
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
	c.books.Reset()
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
