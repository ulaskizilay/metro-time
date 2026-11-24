package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	ledDestWidth = 26
	ledTimeWidth = 8
)

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

// titleCaser applies Turkish title casing (i -> İ, ı -> I).
var titleCaser = cases.Title(language.Turkish)

// directionText turns "A-->>B" into "A → B".
func directionText(direction string) string {
	return strings.ReplaceAll(direction, "-->>", " → ")
}

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
		return "\n" + m.list.View() + "\n" + dimStyle.Render("↑↓: Gez • Enter: Seç • Esc: Geri • Q: Çık")
	}

	var b strings.Builder
	b.WriteString(dimStyle.Render(fmt.Sprintf("%s • %s", m.selectedLine.Code, m.selectedStation.Text)) + "\n")
	b.WriteString(dimStyle.Render(directionText(m.selectedDirection.Text)) + "\n\n")

	switch {
	case m.loading && len(m.trips) == 0:
		b.WriteString("Seferler Yükleniyor...\n")
	case m.errMsg != "" && len(m.trips) == 0:
		b.WriteString(m.errMsg + "\n")
	default:
		b.WriteString(ledPanelStyle.Render(m.ledRows()) + "\n")
		if m.errMsg != "" {
			b.WriteString(dimStyle.Render(m.errMsg) + "\n")
		}
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

func ledText(s string) string {
	r := []rune(titleCaser.String(s))
	if len(r) > ledDestWidth {
		return string(r[:ledDestWidth])
	}
	return string(r)
}
