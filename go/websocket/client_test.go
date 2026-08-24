package websocket

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeDialer struct{ conn *fakeConn }

func (d fakeDialer) DialContext(context.Context, string, http.Header) (Connection, *http.Response, error) {
	return d.conn, nil, nil
}

type fakeRead struct {
	data []byte
	err  error
}
type fakeConn struct {
	mu            sync.Mutex
	reads         chan fakeRead
	writes        [][]byte
	closed        chan struct{}
	once          sync.Once
	pong          func(string) error
	writeDeadline time.Time
}

func newFakeConn() *fakeConn {
	return &fakeConn{reads: make(chan fakeRead, 8), closed: make(chan struct{})}
}
func (c *fakeConn) ReadMessage() (int, []byte, error) {
	select {
	case r := <-c.reads:
		return 1, r.data, r.err
	case <-c.closed:
		return 0, nil, errors.New("closed")
	}
}
func (c *fakeConn) WriteMessage(_ int, b []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writes = append(c.writes, append([]byte(nil), b...))
	return nil
}
func (c *fakeConn) WriteControl(int, []byte, time.Time) error { return nil }
func (c *fakeConn) SetReadDeadline(time.Time) error           { return nil }
func (c *fakeConn) SetWriteDeadline(deadline time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeDeadline = deadline
	return nil
}
func (c *fakeConn) SetPongHandler(h func(string) error) { c.pong = h }
func (c *fakeConn) Close() error                        { c.once.Do(func() { close(c.closed) }); return nil }

func TestClientLifecycleAndRouting(t *testing.T) {
	t.Parallel()
	conn := newFakeConn()
	c, err := NewClient(&ClientOption{EndpointURL: "ws://test", Dialer: fakeDialer{conn}, QueueSize: 4, PingPeriod: 0})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err = c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err = c.Subscribe(ctx, []Channel{{Name: ChannelHeartbeat}}, []string{"BTC-USD"}); err != nil {
		t.Fatal(err)
	}
	if err = c.Unsubscribe(ctx, []Channel{{Name: ChannelHeartbeat}}, []string{"BTC-USD"}); err != nil {
		t.Fatal(err)
	}
	conn.mu.Lock()
	writes := len(conn.writes)
	conn.mu.Unlock()
	if writes != 2 {
		t.Fatalf("writes=%d", writes)
	}
	conn.reads <- fakeRead{data: []byte(`{"type":"heartbeat","product_id":"BTC-USD","sequence":1,"last_trade_id":2,"time":"x"}`)}
	select {
	case e := <-c.Events():
		if _, ok := e.Value.(*Heartbeat); !ok {
			t.Fatalf("event=%T", e.Value)
		}
	case <-time.After(time.Second):
		t.Fatal("event timeout")
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
	if err = c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestContextCancellationClosesConnection(t *testing.T) {
	t.Parallel()
	conn := newFakeConn()
	c, _ := NewClient(&ClientOption{EndpointURL: "ws://test", Dialer: fakeDialer{conn}, QueueSize: 1, PingPeriod: 0})
	ctx, cancel := context.WithCancel(context.Background())
	if err := c.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	cancel()
	select {
	case <-conn.closed:
	case <-time.After(time.Second):
		t.Fatal("connection was not closed")
	}
	_ = c.Close()
}

func TestSessionEndRemainsObservableWhenErrorQueueIsFull(t *testing.T) {
	t.Parallel()
	conn := newFakeConn()
	c, err := NewClient(&ClientOption{EndpointURL: "ws://test", Dialer: fakeDialer{conn}, QueueSize: 1, PingPeriod: 0})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	c.report(errors.New("non-terminal decode error"))
	conn.reads <- fakeRead{err: errors.New("connection lost")}
	select {
	case sessionErr := <-c.SessionEnds():
		if sessionErr == nil {
			t.Fatal("expected session termination error")
		}
	case <-time.After(time.Second):
		t.Fatal("session termination was not reported")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestSlowConsumerTerminatesSession(t *testing.T) {
	t.Parallel()
	conn := newFakeConn()
	c, err := NewClient(&ClientOption{EndpointURL: "ws://test", Dialer: fakeDialer{conn}, QueueSize: 1, PingPeriod: 0})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	for index := 1; index <= 2; index++ {
		conn.reads <- fakeRead{data: []byte(fmt.Sprintf(`{"type":"heartbeat","product_id":"BTC-USD","sequence":%d,"last_trade_id":2,"time":"x"}`, index))}
	}
	select {
	case sessionErr := <-c.SessionEnds():
		if sessionErr == nil || !strings.Contains(sessionErr.Error(), "consumer=too_slow queue_size=1") {
			t.Fatalf("session error = %v", sessionErr)
		}
	case <-time.After(time.Second):
		t.Fatal("slow consumer did not terminate the session")
	}
	select {
	case <-conn.closed:
	case <-time.After(time.Second):
		t.Fatal("slow consumer did not close the connection")
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestDefaultClientOptionUsesExpandedQueue(t *testing.T) {
	t.Parallel()
	if got, want := DefaultClientOption().QueueSize, 1024; got != want {
		t.Fatalf("queue size = %d, want %d", got, want)
	}
}

func TestSubscribeUsesContextWriteDeadline(t *testing.T) {
	t.Parallel()
	conn := newFakeConn()
	c, err := NewClient(&ClientOption{EndpointURL: "ws://test", Dialer: fakeDialer{conn}, QueueSize: 1, PingPeriod: 0})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(time.Second)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	if err := c.Subscribe(ctx, []Channel{{Name: ChannelHeartbeat}}, []string{"BTC-USD"}); err != nil {
		t.Fatal(err)
	}
	conn.mu.Lock()
	got := conn.writeDeadline
	conn.mu.Unlock()
	if !got.Equal(deadline) {
		t.Fatalf("write deadline = %s, want %s", got, deadline)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestNilDialerAndSubscriptionValidation(t *testing.T) {
	t.Parallel()
	if _, err := NewClient(&ClientOption{EndpointURL: "x"}); err == nil {
		t.Fatal("expected nil dialer")
	}
	conn := newFakeConn()
	c, _ := NewClient(&ClientOption{EndpointURL: "x", Dialer: fakeDialer{conn}})
	if err := c.Subscribe(context.Background(), nil, nil); err == nil {
		t.Fatal("expected channels validation")
	}
}
