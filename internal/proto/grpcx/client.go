package grpcx

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type EventType string

const (
	TypeEvent    EventType = "EVENT"
	TypeStatus   EventType = "STATUS"
	TypeError    EventType = "ERROR"
	TypeMetadata EventType = "METADATA"
)

type GRPCEvent struct {
	Timestamp time.Time
	Elapsed   time.Duration
	Direction int // 1: TX, 2: RX, 3: Control/Metadata, 4: Error
	Subtype   string
	Payload   string
	Size      int
	Status    string
	Error     error
}

type Client struct {
	mu        sync.Mutex
	conn      *grpc.ClientConn
	target    string
	eventChan chan GRPCEvent
	closed    bool
}

func NewClient(target string) *Client {
	return &Client{
		target:    target,
		eventChan: make(chan GRPCEvent, 1000),
	}
}

func (c *Client) Events() <-chan GRPCEvent {
	return c.eventChan
}

func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	conn, err := grpc.DialContext(ctx, c.target, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		c.eventChan <- GRPCEvent{
			Timestamp: time.Now(),
			Direction: 4,
			Subtype:   "connect_error",
			Payload:   err.Error(),
			Error:     err,
		}
		return err
	}
	c.conn = conn
	c.eventChan <- GRPCEvent{
		Timestamp: time.Now(),
		Direction: 3,
		Subtype:   "open",
		Payload:   "Connected to gRPC target " + c.target,
	}
	return nil
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return
	}
	c.closed = true
	if c.conn != nil {
		c.conn.Close()
	}
	close(c.eventChan)
}

// UnaryCall invokes a unary gRPC method
func (c *Client) UnaryCall(ctx context.Context, serviceMethod string, jsonPayload string, md metadata.MD) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()

	if conn == nil {
		return errors.New("not connected")
	}

	now := time.Now()
	c.eventChan <- GRPCEvent{
		Timestamp: now,
		Direction: 1, // TX
		Subtype:   "unary_request",
		Payload:   jsonPayload,
		Size:      len(jsonPayload),
	}

	if md.Len() > 0 {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}

	var header, trailer metadata.MD
	var reply json.RawMessage

	// Generic gRPC invocation via Invoke
	err := conn.Invoke(ctx, serviceMethod, []byte(jsonPayload), &reply,
		grpc.Header(&header), grpc.Trailer(&trailer))

	elapsed := time.Since(now)

	if header.Len() > 0 {
		hJSON, _ := json.Marshal(header)
		c.eventChan <- GRPCEvent{
			Timestamp: time.Now(),
			Elapsed:   elapsed,
			Direction: 3,
			Subtype:   "header",
			Payload:   string(hJSON),
		}
	}

	st, _ := status.FromError(err)
	statusStr := "OK"
	if err != nil {
		statusStr = st.Code().String()
		c.eventChan <- GRPCEvent{
			Timestamp: time.Now(),
			Elapsed:   elapsed,
			Direction: 4,
			Subtype:   "error",
			Payload:   err.Error(),
			Status:    statusStr,
			Error:     err,
		}
		return err
	}

	respStr := string(reply)
	if respStr == "" {
		respStr = "{\"status\":\"OK\"}"
	}

	c.eventChan <- GRPCEvent{
		Timestamp: time.Now(),
		Elapsed:   elapsed,
		Direction: 2, // RX
		Subtype:   "unary_response",
		Payload:   respStr,
		Size:      len(respStr),
		Status:    statusStr,
	}

	return nil
}
