package metro

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const (
	defaultBaseURL   = "https://www.metro.istanbul"
	timetablePath    = "/SeferDurumlari/SeferDetaylari"
	ajaxTripsPath    = "/SeferDurumlari/AJAXSeferGetir"
	maxResponseBytes = 4 << 20 // 4 MiB guard against unexpected huge payloads.

	// modeNow is the "right now" query mode; enough for every stop.
	modeNow = "1"

	userAgent      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	acceptLanguage = "tr-TR,tr;q=0.9,en-US;q=0.8,en;q=0.7"
)

var (
	codeRegex = regexp.MustCompile(`formData\.append\(\s*["']kod["']\s*,\s*["']([^"']+)["']\s*\)`)

	// ErrCodeNotFound is returned when the page layout changed.
	ErrCodeNotFound = errors.New("auth code not found on page")
	// ErrMismatch is returned when route and station are not on the same line.
	ErrMismatch = errors.New("station and direction do not match")
)

// Trip is a single departure record.
type Trip struct {
	Station string `json:"istasyon"`
	Line    string `json:"hat"`
	From    string `json:"durak1"`
	To      string `json:"durak2"`
	Time    string `json:"zaman"`
	Day     int    `json:"gun"`
}

type tripsResponse struct {
	Status string `json:"durum"`
	Info   string `json:"bilgi"`
	Trips  []Trip `json:"sefer"`
}

// Client talks to Metro Istanbul. Do not use the zero value; build with NewClient.
type Client struct {
	http    *http.Client
	baseURL string
}

// NewClient builds a client with a 20 second timeout.
func NewClient() *Client {
	return NewClientWithBaseURL(nil, defaultBaseURL)
}

// NewClientWithBaseURL builds a client using the given HTTP client and base URL.
// If httpClient is nil, a default client with a 20 second timeout is used.
func NewClientWithBaseURL(httpClient *http.Client, baseURL string) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 20 * time.Second}
	}
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	return &Client{
		http:    httpClient,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

func setCommonHeaders(req *http.Request) {
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Language", acceptLanguage)
}

// fetchSession opens the timetable page and returns the session cookie
// and the code token. The code changes on every page load,
// so it is deliberately never cached.
func (c *Client) fetchSession(ctx context.Context) (code, cookie string, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+timetablePath, nil)
	if err != nil {
		return "", "", err
	}
	setCommonHeaders(req)

	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	parts := make([]string, 0, len(resp.Cookies()))
	for _, ck := range resp.Cookies() {
		parts = append(parts, ck.Name+"="+ck.Value)
	}
	cookie = strings.Join(parts, "; ")

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return "", "", err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return "", "", err
	}
	doc.Find("script").Each(func(_ int, s *goquery.Selection) {
		if code != "" {
			return
		}
		if m := codeRegex.FindStringSubmatch(s.Text()); m != nil {
			code = m[1]
		}
	})
	if code == "" {
		return "", "", ErrCodeNotFound
	}
	return code, cookie, nil
}

// GetTrips returns departures for the given route (direction) and station.
// Route and station must belong to the same line.
func (c *Client) GetTrips(ctx context.Context, routeID, stationID string) ([]Trip, error) {
	if routeID == "" || stationID == "" {
		return nil, errors.New("route and station must not be empty")
	}

	code, cookie, err := c.fetchSession(ctx)
	if err != nil {
		return nil, fmt.Errorf("session failed: %w", err)
	}

	today := time.Now().Format("02.01.2006")

	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	fields := [][2]string{
		{"secim", modeNow},
		{"tarih2", today},
		{"station", stationID},
		{"route", routeID},
		{"kod", code},
	}
	for _, f := range fields {
		if err := writer.WriteField(f[0], f[1]); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+ajaxTripsPath, &form)
	if err != nil {
		return nil, err
	}
	setCommonHeaders(req)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", c.baseURL)
	req.Header.Set("Referer", c.baseURL+timetablePath)
	req.Header.Set("Cookie", cookie)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("trips request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("trips request returned status %d", resp.StatusCode)
	}

	var answer tripsResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, maxResponseBytes)).Decode(&answer); err != nil {
		return nil, fmt.Errorf("unreadable response: %w", err)
	}
	if answer.Status == "-1" {
		if answer.Info != "" {
			return nil, fmt.Errorf("%w: %s", ErrMismatch, answer.Info)
		}
		return nil, ErrMismatch
	}
	return answer.Trips, nil
}
