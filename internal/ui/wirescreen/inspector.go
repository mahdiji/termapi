package wirescreen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
)

func formatInspector(ev *Event, width, height int) []string {
	if ev == nil {
		return []string{"(no message selected)"}
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("ID: #%d | Dir: %v | Size: %dB", ev.ID, ev.Direction, ev.Size))
	lines = append(lines, fmt.Sprintf("Time: %s", ev.Timestamp.Format("15:04:05.000")))
	if ev.RTT > 0 {
		lines = append(lines, fmt.Sprintf("RTT: %v", ev.RTT))
	}
	lines = append(lines, strings.Repeat("─", max(5, width)))
	lines = append(lines, "Payload:")

	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(ev.Payload), "", "  "); err == nil {
		for _, l := range strings.Split(prettyJSON.String(), "\n") {
			lines = append(lines, l)
		}
	} else if ev.Subtype == "binary" || isBinaryPayload(ev.Payload) {
		for _, l := range strings.Split(hexDump([]byte(ev.Payload), 8), "\n") {
			if l != "" {
				lines = append(lines, l)
			}
		}
	} else {
		for _, l := range strings.Split(ev.Payload, "\n") {
			lines = append(lines, l)
		}
	}

	return lines
}

func isBinaryPayload(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < 32 && s[i] != '\n' && s[i] != '\r' && s[i] != '\t' {
			return true
		}
	}
	return false
}

func hexDump(data []byte, perLine int) string {
	var sb strings.Builder
	for i := 0; i < len(data); i += perLine {
		end := i + perLine
		if end > len(data) {
			end = len(data)
		}
		chunk := data[i:end]

		sb.WriteString(fmt.Sprintf("%04x  ", i))

		var hexPart strings.Builder
		var asciiPart strings.Builder
		for j, b := range chunk {
			hexPart.WriteString(fmt.Sprintf("%02x ", b))
			if j == 7 {
				hexPart.WriteString(" ")
			}
			if b >= 32 && b <= 126 {
				asciiPart.WriteByte(b)
			} else {
				asciiPart.WriteByte('.')
			}
		}
		padding := (perLine - len(chunk)) * 3
		if len(chunk) <= 7 {
			padding += 3
		}
		sb.WriteString(hexPart.String() + strings.Repeat(" ", padding) + " |" + asciiPart.String() + "|\n")
	}
	return sb.String()
}
