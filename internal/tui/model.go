package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ulask/metro-time/internal/metro"
)

const refreshInterval = 30 * time.Second

const (
	defaultListWidth  = 64
	defaultListHeight = 22
	minListHeight     = 14
)

type stage int

const (
	stageWelcome stage = iota
	stageLine
	stageStation
	stageDirection
	stageResult
)

// listItem must expose Title/Description: that is the interface
// DefaultDelegate renders, anything else shows up as blank rows.
type listItem struct {
	itemTitle string
	itemDesc  string
}

func (l listItem) Title() string       { return l.itemTitle }
func (l listItem) Description() string { return l.itemDesc }
func (l listItem) FilterValue() string { return l.itemTitle }

type tripsResultMsg struct {
	trips []metro.Trip
	err   error
	seq   int
}

type refreshTickMsg time.Time

type Model struct {
	lines         []metro.Line
	client        *metro.Client
	totalStations int

	stage             stage
	list              list.Model
	selectedLine      *metro.Line
	selectedStation   metro.Station
	selectedDirection metro.Route
	trips             []metro.Trip
	loading           bool
	errMsg            string
	tripSeq           int
}

func New(lines []metro.Line) Model {
	return Model{
		lines:         lines,
		client:        metro.NewClient(),
		totalStations: countStations(lines),
		stage:         stageWelcome,
		list:          newList(lineTitle(len(lines)), lineChoices(lines), 0, 0),
	}
}

func countStations(lines []metro.Line) int {
	total := 0
	for i := range lines {
		total += len(lines[i].Stations)
	}
	return total
}

func lineChoices(lines []metro.Line) []listItem {
	choices := make([]listItem, 0, len(lines))
	for i := range lines {
		choices = append(choices, listItem{
			itemTitle: lines[i].Code + " — " + lines[i].Name,
			itemDesc:  fmt.Sprintf("%d Durak", len(lines[i].Stations)),
		})
	}
	return choices
}

func stationChoices(line *metro.Line) []listItem {
	choices := make([]listItem, 0, len(line.Stations))
	for _, s := range line.Stations {
		choices = append(choices, listItem{itemTitle: s.Text})
	}
	return choices
}

func directionChoices(line *metro.Line, station metro.Station) []listItem {
	routes := line.RoutesFor(station)
	choices := make([]listItem, 0, len(routes))
	for _, d := range routes {
		choices = append(choices, listItem{itemTitle: directionText(d.Text)})
	}
	return choices
}

func toItems(choices []listItem) []list.Item {
	items := make([]list.Item, 0, len(choices))
	for _, c := range choices {
		items = append(items, c)
	}
	return items
}

func lineTitle(n int) string { return fmt.Sprintf("Hat Seç (%d)", n) }

func stationTitle(line *metro.Line) string {
	return fmt.Sprintf("Durak Seç — %s (%d)", line.Code, len(line.Stations))
}

func directionTitle(station metro.Station) string {
	return "Yön Seç — " + station.Text
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.list.SetSize(listWidth(msg.Width), listHeight(msg.Height))

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			return m.goBack()
		case "r", "R":
			if m.stage == stageResult {
				return m.refreshTrips()
			}
		case "enter":
			if m.stage == stageWelcome {
				m.stage = stageLine
				return m, nil
			}
			return m.chooseSelected()
		}

	case tripsResultMsg:
		if msg.seq != m.tripSeq {
			return m, nil // stale answer from a previous screen
		}
		m.loading = false
		if msg.err != nil {
			m.errMsg = "Bağlantı Hatası, Tekrar Deneniyor..."
			m.trips = nil
		} else {
			m.errMsg = ""
			m.trips = msg.trips
			if len(m.trips) > 2 {
				m.trips = m.trips[:2]
			}
		}
		return m, waitRefresh()

	case refreshTickMsg:
		if m.stage != stageResult {
			return m, nil // user already left; do not fetch in background
		}
		return m.refreshTrips()
	}

	if m.stage != stageResult {
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	}
	return m, nil
}

func listWidth(windowWidth int) int {
	if windowWidth <= 0 {
		return defaultListWidth
	}
	return windowWidth
}

func listHeight(windowHeight int) int {
	if h := windowHeight - 4; h >= minListHeight {
		return h
	}
	return defaultListHeight
}

// chooseSelected resolves the highlight by position: choices always mirror
// the line's own slices, so no search is needed.
func (m Model) chooseSelected() (tea.Model, tea.Cmd) {
	idx := m.list.Index()
	switch m.stage {
	case stageLine:
		if idx < 0 || idx >= len(m.lines) {
			return m, nil
		}
		m.selectedLine = &m.lines[idx]
		m.setList(stationTitle(m.selectedLine), stationChoices(m.selectedLine))
		m.stage = stageStation

	case stageStation:
		if m.selectedLine == nil || idx < 0 || idx >= len(m.selectedLine.Stations) {
			return m, nil
		}
		m.selectedStation = m.selectedLine.Stations[idx]
		m.setList(directionTitle(m.selectedStation), directionChoices(m.selectedLine, m.selectedStation))
		m.stage = stageDirection

	case stageDirection:
		if m.selectedLine == nil {
			return m, nil
		}
		routes := m.selectedLine.RoutesFor(m.selectedStation)
		if idx < 0 || idx >= len(routes) {
			return m, nil
		}
		m.selectedDirection = routes[idx]
		m.stage = stageResult
		return m.refreshTrips()
	}
	return m, nil
}

func (m Model) goBack() (tea.Model, tea.Cmd) {
	switch m.stage {
	case stageLine:
		m.stage = stageWelcome
		m.setList(lineTitle(len(m.lines)), lineChoices(m.lines))
	case stageStation:
		m.stage = stageLine
		m.selectedLine = nil
		m.setList(lineTitle(len(m.lines)), lineChoices(m.lines))
	case stageDirection:
		m.stage = stageStation
		m.setList(stationTitle(m.selectedLine), stationChoices(m.selectedLine))
	case stageResult:
		m.stage = stageDirection
		m.trips = nil
		m.errMsg = ""
		m.tripSeq++ // invalidate in-flight answers
		m.setList(directionTitle(m.selectedStation), directionChoices(m.selectedLine, m.selectedStation))
	default:
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) setList(title string, choices []listItem) {
	m.list.Title = title
	m.list.SetItems(toItems(choices))
	m.list.Select(0)
}

func newList(title string, choices []listItem, windowWidth, windowHeight int) list.Model {
	l := list.New(toItems(choices), list.NewDefaultDelegate(), listWidth(windowWidth), listHeight(windowHeight))
	l.Title = title
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	return l
}

func (m Model) refreshTrips() (tea.Model, tea.Cmd) {
	m.loading = true
	m.errMsg = ""
	m.tripSeq++
	seq := m.tripSeq
	routeID, stationID := m.selectedDirection.ID, m.selectedStation.ID
	client := m.client
	return m, func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		trips, err := client.GetTrips(ctx, routeID, stationID)
		return tripsResultMsg{trips: trips, err: err, seq: seq}
	}
}

func waitRefresh() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg { return refreshTickMsg(t) })
}

// directionText turns "A-->>B" into "A → B".
func directionText(direction string) string {
	return strings.ReplaceAll(direction, "-->>", " → ")
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	welcomeBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("12")).
			Padding(1, 6).
			Align(lipgloss.Center)

	ledPanelStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("0")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("8")).
			Padding(1, 2)
	ledDestStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("10")).
			Background(lipgloss.Color("0"))
	ledTimeStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("202")).
			Background(lipgloss.Color("0"))
	onboardLedStyle = lipgloss.NewStyle().Bold(true).
			Foreground(lipgloss.Color("11")).
			Background(lipgloss.Color("0"))
)

const (
	ledDestWidth = 26
	ledTimeWidth = 8
)

func (m Model) welcomeView() string {
	body := titleStyle.Render("Metro-Time") + "\n\n" +
		"Metro İstanbul Sefer Saatleri Terminalde\n" +
		dimStyle.Render(fmt.Sprintf("%d Hat • %d Durak", len(m.lines), m.totalStations)) + "\n\n" +
		"Hat → Durak → Yön Seç, Sonraki Seferleri Gör\n\n" +
		dimStyle.Render("Enter: Başla • Q: Çık")
	return "\n" + welcomeBoxStyle.Render(body) + "\n"
}

func (m Model) View() string {
	if m.stage == stageWelcome {
		return m.welcomeView()
	}
	if m.stage != stageResult {
		return "\n" + m.list.View() + "\n" + dimStyle.Render("↑↓: Gez • Enter: Seç • Q: Çık")
	}

	var b strings.Builder
	b.WriteString(dimStyle.Render(fmt.Sprintf("%s • %s", m.selectedLine.Code, m.selectedStation.Text)) + "\n")
	b.WriteString(dimStyle.Render(directionText(m.selectedDirection.Text)) + "\n\n")

	switch {
	case m.loading && len(m.trips) == 0:
		b.WriteString("Seferler Yükleniyor...\n")
	case m.errMsg != "":
		b.WriteString(m.errMsg + "\n")
	default:
		b.WriteString(ledPanelStyle.Render(m.ledRows()) + "\n")
	}

	b.WriteString("\n" + dimStyle.Render("R: Yenile • Esc: Geri • Q: Çık"))
	return b.String()
}

func (m Model) ledRows() string {
	if len(m.trips) == 0 {
		return dimStyle.Background(lipgloss.Color("0")).Render("Yaklaşan Sefer Bulunamadı.")
	}
	rows := make([]string, 0, len(m.trips))
	for _, t := range m.trips {
		remaining, onboard := remainingTime(t.Time)
		rendered := ledTimeStyle.Render(remaining)
		if onboard {
			rendered = onboardLedStyle.Render(remaining)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.PlaceHorizontal(ledDestWidth, lipgloss.Left, ledDestStyle.Render(ledText(t.To))),
			lipgloss.PlaceHorizontal(ledTimeWidth, lipgloss.Right, rendered),
		))
	}
	return strings.Join(rows, "\n")
}

// titleCaser applies Turkish title casing (i → İ, ı → I).
var titleCaser = cases.Title(language.Turkish)

func ledText(s string) string {
	r := []rune(titleCaser.String(s))
	if len(r) > ledDestWidth {
		return string(r[:ledDestWidth])
	}
	return string(r)
}

func remainingTime(clock string) (string, bool) {
	hour, min, ok := parseClock(clock)
	if !ok {
		return clock, false
	}

	now := time.Now()
	nowTotal := now.Hour()*60 + now.Minute()
	targetTotal := hour*60 + min
	// Trips after midnight belong to the previous evening's service.
	if targetTotal < nowTotal && now.Hour() > 20 && hour < 4 {
		targetTotal += 24 * 60
	}
	if diff := targetTotal - nowTotal; diff > 0 {
		return fmt.Sprintf("%d Dk", diff), false
	}
	return "Peronda", true
}

func parseClock(clock string) (hour, min int, ok bool) {
	parts := strings.Split(clock, ":")
	if len(parts) != 2 {
		return 0, 0, false
	}
	h, errH := strconv.Atoi(strings.TrimSpace(parts[0]))
	m, errM := strconv.Atoi(strings.TrimSpace(parts[1]))
	if errH != nil || errM != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, false
	}
	return h, m, true
}
