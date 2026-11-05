package wirescreen

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"api/internal/model"
	"api/internal/proto/grpcx"
	"api/internal/proto/ws"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
)

type WsEventMsg ws.WSEvent
type GrpcEventMsg grpcx.GRPCEvent

type ProtocolType string

const (
	ProtoWS   ProtocolType = "WS"
	ProtoGRPC ProtocolType = "gRPC"
)

// FocusArea menentukan panel mana yang menerima input keyboard.
type FocusArea int

const (
	FocusLog FocusArea = iota
	FocusComposer
	FocusInspector
)

type Model struct {
	State            *model.SharedState
	Protocol         ProtocolType
	URL              string
	Client           *ws.Client
	GrpcClient       *grpcx.Client
	Ring             *RingBuffer
	Composer         textarea.Model
	Width            int
	Height           int
	Scroll           int
	AutoFollow       bool
	Connected        bool
	StatusText       string
	LatestTick       string
	Err              error
	InspectorOpen    bool
	AbsoluteTime     bool
	ThroughputSpark  *series
	RttSpark         *series
	msgCountInterval int
	FilterQuery      string
	FilterActive     bool
	ScrubIndex       int
	Scrubbing        bool
	ExportStatus     string

	Focus       FocusArea
	StartedAt   time.Time
	LastThrough float64
	LastRTT     time.Duration
}

func New(state *model.SharedState, url string) *Model {
	ta := textarea.New()
	ta.Placeholder = "Compose message (Ctrl+Enter to send)..."
	ta.SetHeight(3)
	ta.Focus()

	proto := ProtoWS
	if strings.HasPrefix(url, "grpc://") || strings.Contains(url, ":50051") {
		proto = ProtoGRPC
	}

	return &Model{
		State:           state,
		Protocol:        proto,
		URL:             url,
		Ring:            NewRingBuffer(10000),
		Composer:        ta,
		AutoFollow:      true,
		StatusText:      "DISCONNECTED",
		LatestTick:      "(waiting for live feed...)",
		ThroughputSpark: newSeries(30),
		RttSpark:        newSeries(30),
		ScrubIndex:      -1,
		Focus:           FocusComposer,
		StartedAt:       time.Now(),
	}
}

// EditorFocused melaporkan apakah panel composer sedang menerima input.
func (m *Model) EditorFocused() bool {
	return m != nil && m.Focus == FocusComposer
}

// SetSize menyesuaikan lebar/tinggi composer dengan ukuran panel yang tersedia.
func (m *Model) SetSize(width, height int) {
	m.Width = width
	m.Height = height
	composerWidth := max(20, width-6)
	composerHeight := max(4, height/3)
	m.Composer.SetWidth(composerWidth)
	m.Composer.SetHeight(composerHeight)
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.connectCmd(),
		m.tickCmd(),
	)
}

func (m *Model) connectCmd() tea.Cmd {
	return func() tea.Msg {
		if m.Protocol == ProtoWS {
			if m.URL == "" {
				m.URL = "ws://localhost:8080/ws"
			}
			c := ws.NewClient(m.URL, make(http.Header))
			m.Client = c
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := c.Connect(ctx)
			if err != nil {
				return WsEventMsg{
					Timestamp: time.Now(),
					Direction: 4,
					Subtype:   "error",
					Payload:   err.Error(),
				}
			}
			return WsEventMsg{
				Timestamp: time.Now(),
				Direction: 3,
				Subtype:   "status",
				Payload:   "CONNECTED",
			}
		} else {
			if m.URL == "" {
				m.URL = "localhost:50051"
			}
			gc := grpcx.NewClient(m.URL)
			m.GrpcClient = gc
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := gc.Connect(ctx)
			if err != nil {
				return GrpcEventMsg{
					Timestamp: time.Now(),
					Direction: 4,
					Subtype:   "error",
					Payload:   err.Error(),
				}
			}
			return GrpcEventMsg{
				Timestamp: time.Now(),
				Direction: 3,
				Subtype:   "status",
				Payload:   "CONNECTED",
			}
		}
	}
}

func waitForWsEvent(client *ws.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return nil
		}
		ev, ok := <-client.Events()
		if !ok {
			return nil
		}
		return WsEventMsg(ev)
	}
}

func waitForGrpcEvent(client *grpcx.Client) tea.Cmd {
	return func() tea.Msg {
		if client == nil {
			return nil
		}
		ev, ok := <-client.Events()
		if !ok {
			return nil
		}
		return GrpcEventMsg(ev)
	}
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(1*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) exportSession() {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".termapi", "sessions")
	_ = os.MkdirAll(dir, 0755)
	filename := filepath.Join(dir, fmt.Sprintf("session-%s.jsonl", time.Now().Format("20060102-150405")))

	file, err := os.Create(filename)
	if err != nil {
		m.ExportStatus = "Export failed: " + err.Error()
		return
	}
	defer file.Close()

	for _, ev := range m.Ring.Slice() {
		data, _ := json.Marshal(ev)
		file.Write(data)
		file.WriteString("\n")
	}
	m.ExportStatus = "Exported to " + filename
}

type tickMsg time.Time
