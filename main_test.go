package main

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fabioconcina/claumon/internal/api"
	"github.com/fabioconcina/claumon/internal/forecast"
	"github.com/fabioconcina/claumon/internal/server"
)

// fakeForecastStore serves synthetic history: completed sessions for the
// refit and a rising current window for the forecast itself.
type fakeForecastStore struct{ now time.Time }

func (s fakeForecastStore) GetWindowSnapshots(_, _ string, since time.Time) ([]forecast.StoreSnapshot, error) {
	var out []forecast.StoreSnapshot
	for i := 0; i <= 6; i++ {
		t := s.now.Add(time.Duration(i-6) * 10 * time.Minute)
		if !t.Before(since) {
			out = append(out, forecast.StoreSnapshot{Time: t, U: 0.10 + 0.02*float64(i)})
		}
	}
	return out, nil
}

func (s fakeForecastStore) GetCompletedSessions(_ string, _ time.Time, _ int) ([]forecast.StoreSession, error) {
	var out []forecast.StoreSession
	for i, uFinal := range []float64{0.35, 0.50, 0.42, 0.61, 0.28, 0.55} {
		reset := s.now.Add(-time.Duration(i+1) * 5 * time.Hour)
		snaps := make([]forecast.StoreSnapshot, 0, 10)
		for j := 0; j <= 9; j++ {
			snaps = append(snaps, forecast.StoreSnapshot{
				Time: reset.Add(-5*time.Hour + time.Duration(j)*30*time.Minute),
				U:    uFinal * float64(j) / 9,
			})
		}
		out = append(out, forecast.StoreSession{ResetAt: reset, UFinal: uFinal, Snapshots: snaps})
	}
	return out, nil
}

// When the first poll beats the startup refit, its payload has no forecast;
// refreshForecast must attach one as soon as the refit is done, without
// waiting for the next poll.
func TestRefreshForecastAfterStartupRefit(t *testing.T) {
	now := time.Now()
	svc := forecast.NewService(fakeForecastStore{now: now}, forecast.DefaultConfig())
	handlers := server.NewHandlers(t.TempDir(), nil)
	p := &poller{
		broker:   server.NewBroker(),
		handlers: handlers,
		forecast: svc,
		lastUsage: &api.UsageResponse{
			SessionPercent: 22,
			SessionResetAt: now.Add(2 * time.Hour).Format(time.RFC3339),
		},
	}
	forecastAvailable := func() bool {
		w := httptest.NewRecorder()
		handlers.HandleForecast(w, httptest.NewRequest("GET", "/api/forecast", nil))
		var res map[string]interface{}
		json.NewDecoder(w.Body).Decode(&res)
		return res["available"] == true
	}

	p.refreshForecast()
	if forecastAvailable() {
		t.Fatal("forecast available before the refit")
	}

	if !svc.Refit(forecast.GaugeSession, now) {
		t.Fatal("refit failed on the synthetic history")
	}
	p.refreshForecast()
	if !forecastAvailable() {
		t.Fatal("forecast not attached after the refit")
	}
}
