package metro

import "testing"

func mustLoad(t *testing.T) []Line {
	t.Helper()
	lines, err := LoadLines()
	if err != nil {
		t.Fatal(err)
	}
	return lines
}

func findLine(t *testing.T, lines []Line, code string) *Line {
	t.Helper()
	for i := range lines {
		if lines[i].Code == code {
			return &lines[i]
		}
	}
	t.Fatalf("line %s not found", code)
	return nil
}

func findStation(t *testing.T, line *Line, id string) Station {
	t.Helper()
	for _, s := range line.Stations {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("station %s not found on %s", id, line.Code)
	return Station{}
}

func routeIDs(routes []Route) []string {
	ids := make([]string, 0, len(routes))
	for _, r := range routes {
		ids = append(ids, r.ID)
	}
	return ids
}

func equalIDs(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRoutesForTerminus(t *testing.T) {
	m4 := findLine(t, mustLoad(t), "M4")

	// Kadıköy is the first stop: no Kadıköy-bound direction from Kadıköy.
	if got := routeIDs(m4.RoutesFor(findStation(t, m4, "1"))); !equalIDs(got, []string{"70"}) {
		t.Fatalf("Kadıköy routes = %v, want [70]", got)
	}
	// Sabiha is the last stop: only the Kadıköy-bound direction remains.
	if got := routeIDs(m4.RoutesFor(findStation(t, m4, "224"))); !equalIDs(got, []string{"71"}) {
		t.Fatalf("Sabiha routes = %v, want [71]", got)
	}
	// Middle stop: both directions.
	if got := routeIDs(m4.RoutesFor(findStation(t, m4, "8"))); !equalIDs(got, []string{"70", "71"}) {
		t.Fatalf("Bostancı routes = %v, want [70 71]", got)
	}
}

func TestRoutesForBranch(t *testing.T) {
	lines := mustLoad(t)

	// Nurtepe is served by 108,109,110,111; 109 and 110 terminate there.
	m7 := findLine(t, lines, "M7")
	if got := routeIDs(m7.RoutesFor(findStation(t, m7, "149"))); !equalIDs(got, []string{"108", "111"}) {
		t.Fatalf("Nurtepe routes = %v, want [108 111]", got)
	}

	// Seyrantepe branch end: only the Seyrantepe-originating direction.
	m2 := findLine(t, lines, "M2")
	if got := routeIDs(m2.RoutesFor(findStation(t, m2, "31"))); !equalIDs(got, []string{"20"}) {
		t.Fatalf("Seyrantepe routes = %v, want [20]", got)
	}
}

func TestRoutesForUnknownStationFallsBack(t *testing.T) {
	m4 := findLine(t, mustLoad(t), "M4")
	got := routeIDs(m4.RoutesFor(Station{ID: "9999", Text: "Hayali"}))
	if !equalIDs(got, []string{"70", "71"}) {
		t.Fatalf("unknown station routes = %v, want all [70 71]", got)
	}
}

func TestRouteDestination(t *testing.T) {
	if got := RouteDestination("Kadıköy-->>Sabiha Gökçen Havalimanı"); got != "Sabiha Gökçen Havalimanı" {
		t.Fatalf("destination = %q", got)
	}
}
