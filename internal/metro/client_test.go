package metro

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTripsSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case timetablePath:
			http.SetCookie(w, &http.Cookie{Name: "ASP.NET_SessionId", Value: "sess123"})
			w.Write([]byte(`<html><script>formData.append("kod", "secret-token");</script></html>`))
		case ajaxTripsPath:
			if r.Header.Get("Cookie") != "ASP.NET_SessionId=sess123" {
				t.Errorf("cookie = %q, want ASP.NET_SessionId=sess123", r.Header.Get("Cookie"))
			}
			if r.FormValue("kod") != "secret-token" {
				t.Errorf("kod = %q, want secret-token", r.FormValue("kod"))
			}
			if r.FormValue("route") != "70" || r.FormValue("station") != "8" {
				t.Errorf("unexpected route/station: %s/%s", r.FormValue("route"), r.FormValue("station"))
			}
			_ = json.NewEncoder(w).Encode(tripsResponse{
				Status: "1",
				Trips: []Trip{
					{Station: "Bostancı", Line: "M4", To: "Sabiha Gökçen Havalimanı", Time: "12:30"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewClientWithBaseURL(srv.Client(), srv.URL)
	trips, err := client.GetTrips(context.Background(), "70", "8")
	if err != nil {
		t.Fatalf("GetTrips() error = %v", err)
	}
	if len(trips) != 1 || trips[0].Time != "12:30" {
		t.Fatalf("trips = %+v, want 1 trip at 12:30", trips)
	}
}

func TestGetTripsEmptyIDs(t *testing.T) {
	client := NewClientWithBaseURL(nil, "http://127.0.0.1:0")
	if _, err := client.GetTrips(context.Background(), "", "8"); err == nil {
		t.Fatal("expected error for empty routeID")
	}
	if _, err := client.GetTrips(context.Background(), "70", ""); err == nil {
		t.Fatal("expected error for empty stationID")
	}
}

func TestGetTripsSessionHTTPStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := NewClientWithBaseURL(srv.Client(), srv.URL)
	_, err := client.GetTrips(context.Background(), "70", "8")
	if err == nil {
		t.Fatal("expected error on 503 status")
	}
	if errors.Is(err, ErrCodeNotFound) {
		t.Fatalf("503 should not be masked as ErrCodeNotFound: %v", err)
	}
}

func TestGetTripsCodeNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html><script>console.log("no code")</script></html>`))
	}))
	defer srv.Close()

	client := NewClientWithBaseURL(srv.Client(), srv.URL)
	_, err := client.GetTrips(context.Background(), "70", "8")
	if !errors.Is(err, ErrCodeNotFound) {
		t.Fatalf("err = %v, want ErrCodeNotFound", err)
	}
}

func TestGetTripsAjaxHTTPStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == timetablePath {
			w.Write([]byte(`<html><script>formData.append('kod','tok');</script></html>`))
			return
		}
		http.Error(w, "bad gateway", http.StatusBadGateway)
	}))
	defer srv.Close()

	client := NewClientWithBaseURL(srv.Client(), srv.URL)
	if _, err := client.GetTrips(context.Background(), "70", "8"); err == nil {
		t.Fatal("expected error on 502 status")
	}
}

func TestGetTripsMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == timetablePath {
			w.Write([]byte(`<html><script>formData.append('kod','tok');</script></html>`))
			return
		}
		_ = json.NewEncoder(w).Encode(tripsResponse{
			Status: "-1",
			Info:   "hat ve durak uyumsuz",
		})
	}))
	defer srv.Close()

	client := NewClientWithBaseURL(srv.Client(), srv.URL)
	_, err := client.GetTrips(context.Background(), "70", "8")
	if !errors.Is(err, ErrMismatch) {
		t.Fatalf("err = %v, want ErrMismatch", err)
	}
}
