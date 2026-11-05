package wirescreen

import (
	"time"
)

type Direction int

const (
	DirNone Direction = iota
	DirTX
	DirRX
	DirControl
	DirError
)

type EventType string

const (
	TypeData    EventType = "DATA"
	TypeControl EventType = "CONTROL"
	TypeError   EventType = "ERROR"
	TypeStatus  EventType = "STATUS"
)

type Event struct {
	ID            int64
	Timestamp     time.Time
	Elapsed       time.Duration
	Direction     Direction
	Type          EventType
	Subtype       string // e.g. "text", "binary", "ping", "pong", "open", "close"
	Payload       string
	Size          int
	RTT           time.Duration
	CorrelationID int64
	Dropped       int
	Status        string
}

type RingBuffer struct {
	events   []Event
	capacity int
	head     int
	tail     int
	size     int
	total    int64
	dropped  int
}

func NewRingBuffer(capacity int) *RingBuffer {
	if capacity <= 0 {
		capacity = 10000
	}
	return &RingBuffer{
		events:   make([]Event, capacity),
		capacity: capacity,
	}
}

func (rb *RingBuffer) Push(ev Event) {
	rb.total++
	ev.ID = rb.total
	if rb.size < rb.capacity {
		rb.events[rb.tail] = ev
		rb.tail = (rb.tail + 1) % rb.capacity
		rb.size++
	} else {
		rb.events[rb.tail] = ev
		rb.tail = (rb.tail + 1) % rb.capacity
		rb.head = (rb.head + 1) % rb.capacity
		rb.dropped++
	}
	ev.Dropped = rb.dropped
}

func (rb *RingBuffer) Len() int {
	return rb.size
}

func (rb *RingBuffer) Dropped() int {
	return rb.dropped
}

func (rb *RingBuffer) Slice() []Event {
	res := make([]Event, rb.size)
	for i := 0; i < rb.size; i++ {
		idx := (rb.head + i) % rb.capacity
		res[i] = rb.events[idx]
	}
	return res
}

func (rb *RingBuffer) Clear() {
	rb.head = 0
	rb.tail = 0
	rb.size = 0
	rb.total = 0
	rb.dropped = 0
}
