package wirescreen

import (
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"
)

func TestRingBuffer(t *testing.T) {
	rb := NewRingBuffer(3)
	if rb.Len() != 0 {
		t.Fatalf("expected len 0, got %d", rb.Len())
	}

	rb.Push(Event{Payload: "one"})
	rb.Push(Event{Payload: "two"})
	rb.Push(Event{Payload: "three"})

	if rb.Len() != 3 {
		t.Fatalf("expected len 3, got %d", rb.Len())
	}

	// Push 4th event, should drop "one"
	rb.Push(Event{Payload: "four"})
	if rb.Len() != 3 {
		t.Fatalf("expected len 3 after overflow, got %d", rb.Len())
	}
	if rb.Dropped() != 1 {
		t.Fatalf("expected dropped 1, got %d", rb.Dropped())
	}

	slice := rb.Slice()
	if slice[0].Payload != "two" || slice[1].Payload != "three" || slice[2].Payload != "four" {
		t.Fatalf("unexpected slice order: %v", slice)
	}
}

func TestFitLineAnsiWidth(t *testing.T) {
	width := 40
	lines := []string{
		"Hello world",
		"\x1b[31mRed text\x1b[0m normal",
		"Very long line that exceeds the target width significantly and needs truncation",
	}

	for _, l := range lines {
		fitted := fitLine(l, width)
		w := ansi.StringWidth(fitted)
		if w != width {
			t.Errorf("fitLine width assertion failed: got width %d, expected %d for line %q", w, width, l)
		}
	}
}

func TestEventCorrelationAndTiming(t *testing.T) {
	rb := NewRingBuffer(10)
	now := time.Now()

	reqEv := Event{
		ID:            1,
		Timestamp:     now,
		Direction:     DirTX,
		Type:          TypeData,
		Subtype:       "text",
		Payload:       `{"op":"subscribe","id":1}`,
		CorrelationID: 1,
	}
	respEv := Event{
		ID:            2,
		Timestamp:     now.Add(42 * time.Millisecond),
		Elapsed:       42 * time.Millisecond,
		Direction:     DirRX,
		Type:          TypeData,
		Subtype:       "text",
		Payload:       `{"op":"ack","id":1}`,
		CorrelationID: 1,
	}

	rb.Push(reqEv)
	rb.Push(respEv)

	slice := rb.Slice()
	if len(slice) != 2 {
		t.Fatalf("expected 2 events, got %d", len(slice))
	}
	if slice[0].CorrelationID != slice[1].CorrelationID {
		t.Errorf("expected matching correlation IDs, got %d and %d", slice[0].CorrelationID, slice[1].CorrelationID)
	}
	diff := slice[1].Timestamp.Sub(slice[0].Timestamp)
	if diff != 42*time.Millisecond {
		t.Errorf("expected 42ms diff, got %v", diff)
	}
}

func TestFmtRel(t *testing.T) {
	tests := []struct {
		d        time.Duration
		expected string
	}{
		{0, "00:00.000"},
		{1 * time.Millisecond, "00:00.001"},
		{1005 * time.Millisecond, "00:01.005"},
		{75 * time.Second + 1*time.Millisecond, "01:15.001"},
	}

	for _, tc := range tests {
		got := fmtRel(tc.d)
		if got != tc.expected {
			t.Errorf("fmtRel(%v) = %q, expected %q", tc.d, got, tc.expected)
		}
	}
}

func TestDirLabel(t *testing.T) {
	if dirLabel(DirTX) != "TX" {
		t.Errorf("expected TX, got %q", dirLabel(DirTX))
	}
	if dirLabel(DirRX) != "RX" {
		t.Errorf("expected RX, got %q", dirLabel(DirRX))
	}
	if dirLabel(DirControl) != "CTRL" {
		t.Errorf("expected CTRL, got %q", dirLabel(DirControl))
	}
	if dirLabel(DirError) != "ERR" {
		t.Errorf("expected ERR, got %q", dirLabel(DirError))
	}
}

func TestSparklineCases(t *testing.T) {
	// 1. input kosong, width 5 -> "▁▁▁▁▁"
	res1 := sparkline(nil, 5)
	if res1 != "▁▁▁▁▁" || ansi.StringWidth(res1) != 5 {
		t.Errorf("case 1 failed: got %q", res1)
	}

	// 2. input [0,1,2,3,4,5,6,7], width 8 -> "▁▂▃▄▅▆▇█"
	res2 := sparkline([]float64{0, 1, 2, 3, 4, 5, 6, 7}, 8)
	if res2 != "▁▂▃▄▅▆▇█" || ansi.StringWidth(res2) != 8 {
		t.Errorf("case 2 failed: got %q", res2)
	}

	// 3. semua nilai sama -> semua "▁"
	res3 := sparkline([]float64{5, 5, 5, 5}, 4)
	if res3 != "▁▁▁▁" || ansi.StringWidth(res3) != 4 {
		t.Errorf("case 3 failed: got %q", res3)
	}

	// 4. len(vals) > width -> hanya nilai terakhir yang dipakai
	res4 := sparkline([]float64{0, 0, 0, 10, 20, 30, 40}, 3)
	if ansi.StringWidth(res4) != 3 {
		t.Errorf("case 4 width failed: %d", ansi.StringWidth(res4))
	}

	// 5. assert ansi.StringWidth(hasil) == width pada semua kasus
	widths := []int{1, 5, 10, 20, 50}
	for _, w := range widths {
		s := sparkline([]float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}, w)
		if ansi.StringWidth(s) != w {
			t.Errorf("width assertion failed for width %d: got width %d", w, ansi.StringWidth(s))
		}
	}
}
