// metro-time: Metro Istanbul departures in your terminal.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ulask/metro-time/internal/metro"
	"github.com/ulask/metro-time/internal/tui"
)

func main() {
	lines, err := metro.LoadLines()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Hat Listesi Yüklenemedi:", err)
		os.Exit(1)
	}
	if _, err := tea.NewProgram(tui.New(lines), tea.WithAltScreen()).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Uygulama Hatası:", err)
		os.Exit(1)
	}
}
