package ws

import (
	"context"
	"errors"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
)

type Client struct {
	mu          sync.Mutex
	conn        *websocket.Conn
	url         string
	headers     http.Header
	connected   bool
	closeChan   chan struct{}
	eventChan   chan WSEvent
	startTime   time.Time
	lastPing    time.Time
	rtt         time.Duration
}

type WSEvent struct {
	Timestamp     time.Time
	Elapsed       time.Duration
	Direction     int // 1: TX, 2: RX, 3: Control, 4: Error
	Subtype       string
	Payload       string
	Size          int
	RTT           time.Duration
	Error         error
}

func NewClient(url string, headers http.Header) *Client {
	return &Client{
		url:       url,
		headers:   headers,
		closeChan: make(chan struct{}),
		eventChan: make(chan WSEvent, 1000),
	}
}

func (c *Client) Events() <-chan WSEvent {
	return c.eventChan
}

func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	if c.connected {
		c.mu.Unlock()
		return errors.New("already connected")
	}
	c.mu.Unlock()

	opts := &websocket.DialOptions{
		HTTPHeader: c.headers,
	}

	conn, _, err := websocket.Dial(ctx, c.url, opts)
	if err != nil {
		c.eventChan <- WSEvent{
			Timestamp: time.Now(),
			Direction: 4,
			Subtype:   "connect_error",
			Payload:   err.Error(),
			Error:     err,
		}
		return err
	}

	c.mu.Lock()
	c.conn = conn
	c.connected = true
	c.startTime = time.Now()
	c.mu.Unlock()

	c.eventChan <- WSEvent{
		Timestamp: c.startTime,
		Elapsed:   0,
		Direction: 3,
		Subtype:   "open",
		Payload:   "Connected to " + c.url,
	}

	go c.readLoop()
	go c.pingLoop()

	return nil
}

func (c *Client) readLoop() {
	defer c.Disconnect(websocket.StatusNormalClosure, "client disconnect")

	for {
		ctx := context.Background()
		typ, data, err := c.conn.Read(ctx)
		now := time.Now()
		elapsed := now.Sub(c.startTime)

		if err != nil {
			if errors.Is(err, io.EOF) || websocket.CloseStatus(err) != -1 {
				c.eventChan <- WSEvent{
					Timestamp: now,
					Elapsed:   elapsed,
					Direction: 3,
					Subtype:   "close",
					Payload:   err.Error(),
				}
			} else {
				c.eventChan <- WSEvent{
					Timestamp: now,
					Elapsed:   elapsed,
					Direction: 4,
					Subtype:   "read_error",
					Payload:   err.Error(),
					Error:     err,
				}
			}
			return
		}

		sub := "text"
		if typ == websocket.MessageBinary {
			sub = "binary"
		}

		c.eventChan <- WSEvent{
			Timestamp: now,
			Elapsed:   elapsed,
			Direction: 2, // RX
			Subtype:   sub,
			Payload:   string(data),
			Size:      len(data),
		}
	}
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.closeChan:
			return
		case <-ticker.C:
			c.mu.Lock()
			if !c.connected || c.conn == nil {
				c.mu.Unlock()
				return
			}
			c.lastPing = time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := c.conn.Ping(ctx)
			cancel()
			c.mu.Unlock()

			now := time.Now()
			elapsed := now.Sub(c.startTime)
			rtt := now.Sub(c.lastPing)

			if err != nil {
				c.eventChan <- WSEvent{
					Timestamp: now,
					Elapsed:   elapsed,
					Direction: 4,
					Subtype:   "ping_error",
					Payload:   err.Error(),
				}
			} else {
				c.mu.Lock()
				c.rtt = rtt
				c.mu.Unlock()

				c.eventChan <- WSEvent{
					Timestamp: now,
					Elapsed:   elapsed,
					Direction: 3,
					Subtype:   "pong",
					Payload:   "pong",
					RTT:       rtt,
				}
			}
		}
	}
}

func (c *Client) Send(ctx context.Context, text string, isBinary bool) error {
	c.mu.Lock()
	if !c.connected || c.conn == nil {
		c.mu.Unlock()
		return errors.New("not connected")
	}
	conn := c.conn
	c.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(c.startTime)

	typ := websocket.MessageText
	sub := "text"
	if isBinary {
		typ = websocket.MessageBinary
		sub = "binary"
	}

	err := conn.Write(ctx, typ, []byte(text))
	if err != nil {
		c.eventChan <- WSEvent{
			Timestamp: now,
			Elapsed:   elapsed,
			Direction: 4,
			Subtype:   "write_error",
			Payload:   err.Error(),
			Error:     err,
		}
		return err
	}

	c.eventChan <- WSEvent{
		Timestamp: now,
		Elapsed:   elapsed,
		Direction: 1, // TX
		Subtype:   sub,
		Payload:   text,
		Size:      len(text),
	}

	return nil
}

func (c *Client) Disconnect(status websocket.StatusCode, reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}
	c.connected = false
	close(c.closeChan)

	if c.conn != nil {
		c.conn.Close(status, reason)
		c.conn = nil
	}
}

func (c *Client) RTT() time.Duration {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rtt
}

func (c *Client) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connected
}
