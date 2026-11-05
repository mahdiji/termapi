package wirescreen

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.FilterActive {
			switch msg.String() {
			case "enter":
				m.FilterActive = false
				return m, nil
			case "esc":
				m.FilterActive = false
				m.FilterQuery = ""
				return m, nil
			case "backspace":
				if len(m.FilterQuery) > 0 {
					m.FilterQuery = m.FilterQuery[:len(m.FilterQuery)-1]
				}
				return m, nil
			default:
				if len(msg.String()) == 1 {
					m.FilterQuery += msg.String()
				}
				return m, nil
			}
		}

		// Saat editor fokus, hampir semua tombol harus masuk ke textarea.
		// Hanya shortcut khusus berikut yang dicegat.
		if m.Focus == FocusComposer {
			switch msg.String() {
			case "ctrl+enter":
				text := m.Composer.Value()
				if text != "" {
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					defer cancel()
					if m.Protocol == ProtoWS && m.Client != nil {
						_ = m.Client.Send(ctx, text, false)
					} else if m.Protocol == ProtoGRPC && m.GrpcClient != nil {
						_ = m.GrpcClient.UnaryCall(ctx, "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo", text, nil)
					}
					m.Composer.SetValue("")
				}
				return m, nil
			case "tab":
				if m.InspectorOpen {
					m.Focus = FocusInspector
				} else {
					m.Focus = FocusLog
				}
				m.Composer.Blur()
				return m, nil
			case "shift+tab":
				m.Focus = FocusLog
				m.Composer.Blur()
				return m, nil
			case "esc":
				m.Focus = FocusLog
				m.Composer.Blur()
				return m, nil
			}
			m.Composer, cmd = m.Composer.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "ctrl+c", "q":
			if m.Client != nil {
				m.Client.Disconnect(1000, "user exit")
			}
			if m.GrpcClient != nil {
				m.GrpcClient.Close()
			}
			return m, tea.Quit
		case "tab":
			// Siklus fokus: Log -> Composer -> (Inspector) -> Log.
			switch m.Focus {
			case FocusLog:
				m.Focus = FocusComposer
				m.Composer.Focus()
			case FocusComposer:
				if m.InspectorOpen {
					m.Focus = FocusInspector
					m.Composer.Blur()
				} else {
					m.Focus = FocusLog
					m.Composer.Blur()
				}
			default:
				m.Focus = FocusLog
			}
			return m, nil
		case "shift+tab":
			switch m.Focus {
			case FocusLog:
				m.Focus = FocusComposer
				m.Composer.Focus()
			case FocusComposer:
				m.Focus = FocusLog
				m.Composer.Blur()
			default:
				m.Focus = FocusComposer
				m.Composer.Focus()
			}
			return m, nil
		case "i":
			if m.Focus == FocusLog {
				m.InspectorOpen = !m.InspectorOpen
				if m.InspectorOpen && m.Focus == FocusLog {
					m.Focus = FocusInspector
				}
				return m, nil
			}
		case "t":
			m.AbsoluteTime = !m.AbsoluteTime
			return m, nil
		case "ctrl+s":
			m.exportSession()
			return m, nil
		case "/":
			m.FilterActive = true
			return m, nil
		case "[":
			m.Scrubbing = true
			if m.ScrubIndex < 0 {
				m.ScrubIndex = max(0, m.Ring.Len()-1)
			} else if m.ScrubIndex > 0 {
				m.ScrubIndex--
			}
			return m, nil
		case "]":
			m.Scrubbing = true
			if m.ScrubIndex < m.Ring.Len()-1 {
				m.ScrubIndex++
			}
			return m, nil
		case "ctrl+enter":
			text := m.Composer.Value()
			if text != "" {
				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				if m.Protocol == ProtoWS && m.Client != nil {
					_ = m.Client.Send(ctx, text, false)
				} else if m.Protocol == ProtoGRPC && m.GrpcClient != nil {
					_ = m.GrpcClient.UnaryCall(ctx, "/grpc.reflection.v1alpha.ServerReflection/ServerReflectionInfo", text, nil)
				}
				m.Composer.SetValue("")
			}
			return m, nil
		case "up", "k":
			if m.Scroll > 0 {
				m.Scroll--
				m.AutoFollow = false
			}
			return m, nil
		case "down", "j":
			m.Scroll++
			eventsLen := m.Ring.Len()
			if m.Scroll >= eventsLen {
				m.Scroll = max(0, eventsLen-1)
				m.AutoFollow = true
			}
			return m, nil
		case "G":
			m.AutoFollow = true
			m.Scroll = max(0, m.Ring.Len()-1)
			m.Scrubbing = false
			return m, nil
		}

	case WsEventMsg:
		cleanPayload := strings.NewReplacer("\r", "", "\n", " ").Replace(msg.Payload)
		dir := Direction(msg.Direction)
		evType := TypeData
		if msg.Direction == 3 {
			evType = TypeControl
			if msg.Subtype == "status" && msg.Payload == "CONNECTED" {
				m.Connected = true
				m.StatusText = "● OPEN"
				return m, waitForWsEvent(m.Client)
			}
		} else if msg.Direction == 4 {
			evType = TypeError
			m.StatusText = "✕ CLOSED"
		}

		m.msgCountInterval++

		if strings.Contains(cleanPayload, `"tick":`) {
			timeStr := msg.Timestamp.Format("15:04:05.000")
			m.LatestTick = fmt.Sprintf("[%s] ◂ %s", timeStr, cleanPayload)
			return m, waitForWsEvent(m.Client)
		}

		m.Ring.Push(Event{
			Timestamp: msg.Timestamp,
			Elapsed:   msg.Elapsed,
			Direction: dir,
			Type:      evType,
			Subtype:   msg.Subtype,
			Payload:   cleanPayload,
			Size:      msg.Size,
			RTT:       msg.RTT,
		})

		if m.AutoFollow && !m.Scrubbing {
			m.Scroll = max(0, m.Ring.Len()-1)
		}
		return m, waitForWsEvent(m.Client)

	case GrpcEventMsg:
		cleanPayload := strings.NewReplacer("\r", "", "\n", " ").Replace(msg.Payload)
		dir := Direction(msg.Direction)
		evType := TypeData
		if msg.Direction == 3 {
			evType = TypeControl
			if msg.Subtype == "status" && msg.Payload == "CONNECTED" {
				m.Connected = true
				m.StatusText = "● gRPC OPEN"
				return m, waitForGrpcEvent(m.GrpcClient)
			}
		} else if msg.Direction == 4 {
			evType = TypeError
			m.StatusText = "✕ gRPC CLOSED"
		}

		m.msgCountInterval++

		m.Ring.Push(Event{
			Timestamp: msg.Timestamp,
			Elapsed:   msg.Elapsed,
			Direction: dir,
			Type:      evType,
			Subtype:   msg.Subtype,
			Payload:   cleanPayload,
			Size:      msg.Size,
			Status:    msg.Status,
		})

		if m.AutoFollow && !m.Scrubbing {
			m.Scroll = max(0, m.Ring.Len()-1)
		}
		return m, waitForGrpcEvent(m.GrpcClient)

	case tickMsg:
		if m.Protocol == ProtoWS && m.Client != nil {
			if m.Client.IsConnected() {
				m.Connected = true
				m.StatusText = "● OPEN"
			} else {
				m.Connected = false
				m.StatusText = "○ CLOSED"
			}
			rtt := m.Client.RTT()
			if rtt > 0 {
				m.LastRTT = rtt
				m.RttSpark.push(float64(rtt.Milliseconds()))
			}
		} else if m.Protocol == ProtoGRPC && m.GrpcClient != nil {
			m.Connected = true
			m.StatusText = "● gRPC OPEN"
		}
		m.LastThrough = float64(m.msgCountInterval)
		m.ThroughputSpark.push(m.LastThrough)
		m.msgCountInterval = 0

		return m, m.tickCmd()

	case tea.WindowSizeMsg:
		m.SetSize(msg.Width, msg.Height)
		return m, nil
	}

	m.Composer, cmd = m.Composer.Update(msg)
	return m, cmd
}
