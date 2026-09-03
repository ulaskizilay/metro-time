package data

import _ "embed"

// LinesJSON is the embedded line/station catalogue.
// Source: https://www.metro.istanbul/SeferDurumlari/SeferDetaylari
//
//go:embed metro-hatlar.json
var LinesJSON []byte
