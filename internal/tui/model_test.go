package tui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ulask/metro-time/internal/metro"
)

func mustLoad(t *testing.T) []metro.Line {
	t.Helper()
	lines, err := metro.LoadLines()
	if err != nil {
		t.Fatal(err)
	}
	return lines
}

func TestLineListComplete(t *testing.T) {
	m := New(mustLoad(t))
	if got := len(m.list.Items()); got != 18 {
		t.Fatalf("line count = %d, want 18", got)
	}
}

func TestWelcomeScreenFirst(t *testing.T) {
	view := New(mustLoad(t)).View()
	for _, want := range []string{"Metro-Time", "18", "248"} {
		if !strings.Contains(view, want) {
			t.Fatalf("welcome view missing %q, got:\n%s", want, view)
		}
	}
}

func pressEnter(t *testing.T, m Model) Model {
	t.Helper()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	got, ok := updated.(Model)
	if !ok {
		t.Fatal("Update did not return a Model")
	}
	return got
}

func TestFullFlowToDirection(t *testing.T) {
	lines := mustLoad(t)
	m := New(lines)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = updated.(Model)
	m = pressEnter(t, m) // welcome -> line

	// Jump to T1 (31 stations, the longest list).
	t1 := -1
	for i, item := range m.list.Items() {
		if li, ok := item.(listItem); ok && len(li.itemTitle) >= 2 && li.itemTitle[:2] == "T1" {
			t1 = i
		}
	}
	if t1 < 0 {
		t.Fatal("T1 not found")
	}
	m.list.Select(t1)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model) // -> station
	if view := m.View(); !strings.Contains(view, "Kabataş") {
		t.Fatalf("station view missing Kabataş, got:\n%s", view)
	}

	// Pick Kabataş (index 0): only the Kabataş-originating direction remains.
	m.list.Select(0)
	updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model) // -> direction
	view := m.View()
	if !strings.Contains(view, "Bağcılar") {
		t.Fatalf("direction view missing Bağcılar, got:\n%s", view)
	}
	if strings.Contains(view, "Cevizlibağ") {
		t.Fatalf("direction view leaks unserved routes, got:\n%s", view)
	}
}

// The line screen must render rows (not blank) and offer all 18 lines
// across its pages; the first page shows the first lines.
func TestLineScreenShowsManyLines(t *testing.T) {
	lines := mustLoad(t)
	m := New(lines)
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	m = updated.(Model)
	m = pressEnter(t, m)
	view := m.View()
	for _, want := range []string{"F1", "M1A", "M2", "M4"} {
		if !strings.Contains(view, want) {
			t.Fatalf("first page missing %q, got:\n%s", want, view)
		}
	}
	if got := len(m.list.Items()); got != 18 {
		t.Fatalf("total items = %d, want 18", got)
	}
}

func TestRemainingTimePast(t *testing.T) {
	oneHourAgo := time.Now().Add(-time.Hour).Format("15:04")
	text, onboard := remainingTime(oneHourAgo)
	if !onboard || text != "Peronda" {
		t.Fatalf("past time (%s) = %q,%v; want Peronda,true", oneHourAgo, text, onboard)
	}
}

func TestRemainingTimeFuture(t *testing.T) {
	inTwoHours := time.Now().Add(2 * time.Hour).Format("15:04")
	text, onboard := remainingTime(inTwoHours)
	if onboard {
		t.Fatalf("future time (%s) returned onboard=true", inTwoHours)
	}
	var minutes int
	if _, err := fmt.Sscanf(text, "%d Dk", &minutes); err != nil || minutes <= 0 || minutes > 130 {
		t.Fatalf("future time (%s) = %q; unreasonable minutes", inTwoHours, text)
	}
}

func TestStaleAnswerIgnored(t *testing.T) {
	m := New(mustLoad(t))
	m.tripSeq = 5
	updated, _ := m.Update(tripsResultMsg{trips: []metro.Trip{{To: "X"}}, seq: 4})
	got := updated.(Model)
	if len(got.trips) != 0 || got.loading {
		t.Fatalf("stale answer applied: trips=%v loading=%v", got.trips, got.loading)
	}
}

func TestTickOffResultDoesNothing(t *testing.T) {
	m := New(mustLoad(t))
	m.stage = stageDirection
	updated, cmd := m.Update(refreshTickMsg(time.Now()))
	got := updated.(Model)
	if cmd != nil || got.loading {
		t.Fatal("off-result tick started a fetch")
	}
}
func TestLedTextTurkishTitle(t *testing.T) {
	for in, want := range map[string]string{
		"KADIKÖY":                  "Kadıköy",
		"İMES":                     "İmes",
		"Sabiha Gökçen Havalimanı": "Sabiha Gökçen Havalimanı",
	} {
		if got := ledText(in); got != want {
			t.Fatalf("ledText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestResultBoardStyle(t *testing.T) {
	lines := mustLoad(t)
	var m4 *metro.Line
	for i := range lines {
		if lines[i].Code == "M4" {
			m4 = &lines[i]
		}
	}
	if m4 == nil {
		t.Fatal("M4 not found")
	}
	soon := time.Now().Add(5 * time.Minute).Format("15:04")
	m := New(lines)
	m.stage = stageResult
	m.selectedLine = m4
	m.selectedStation = m4.Stations[0]
	m.selectedDirection = m4.Routes[0]
	m.trips = []metro.Trip{{To: "Sabiha Gökçen Havalimanı", Time: soon}}
	view := m.View()
	for _, want := range []string{"Sabiha", "Dk"} {
		if !strings.Contains(view, want) {
			t.Fatalf("board view missing %q, got:\n%s", want, view)
		}
	}
}
