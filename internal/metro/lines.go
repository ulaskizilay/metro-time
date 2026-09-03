package metro

import (
	"encoding/json"
	"strings"

	"github.com/ulask/metro-time/internal/data"
)

// Route is one direction of a line.
type Route struct {
	ID   string `json:"route"`
	Text string `json:"text"`
}

// Station is a single stop.
type Station struct {
	ID   string `json:"station"`
	Text string `json:"text"`
}

// Line is a metro/tram/funicular/cable-car line.
type Line struct {
	Code     string    `json:"hatKodu"`
	Name     string    `json:"hatAdi"`
	HatID    string    `json:"hatId"`
	Routes   []Route   `json:"routes"`
	Stations []Station `json:"stations"`
	// StationRoutes maps a station ID to the route IDs serving it.
	StationRoutes map[string][]string `json:"stationRoutes"`
}

// RoutesFor returns the boardable routes at the given station: the routes
// serving it, minus the ones terminating there (e.g. no Kadıköy-bound
// direction from Kadıköy itself). It never returns empty: on any data gap
// it falls back to the serving routes, then to all routes.
func (l *Line) RoutesFor(station Station) []Route {
	serving := l.Routes
	if ids, ok := l.StationRoutes[station.ID]; ok && len(ids) > 0 {
		allowed := make(map[string]bool, len(ids))
		for _, id := range ids {
			allowed[id] = true
		}
		filtered := make([]Route, 0, len(ids))
		for _, r := range l.Routes {
			if allowed[r.ID] {
				filtered = append(filtered, r)
			}
		}
		serving = filtered
	}
	var boardable []Route
	for _, r := range serving {
		if RouteDestination(r.Text) != station.Text {
			boardable = append(boardable, r)
		}
	}
	if len(boardable) == 0 {
		return serving
	}
	return boardable
}

// RouteDestination returns the text after "-->>" in "A-->>B".
func RouteDestination(routeText string) string {
	parts := strings.Split(routeText, "-->>")
	return strings.TrimSpace(parts[len(parts)-1])
}

type linesFile struct {
	Lines []Line `json:"hatlar"`
}

// LoadLines reads the embedded line/station catalogue.
func LoadLines() ([]Line, error) {
	var file linesFile
	if err := json.Unmarshal(data.LinesJSON, &file); err != nil {
		return nil, err
	}
	return file.Lines, nil
}
