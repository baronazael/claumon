package server

import (
	"log"
	"math"
	"time"

	"github.com/fabioconcina/claumon/internal/limitshare"
	"github.com/fabioconcina/claumon/internal/parser"
)

const (
	sessionWindow = 5 * time.Hour
	weeklyWindow  = 7 * 24 * time.Hour
)

// AttachLimitShares sets each session's estimated share of the 5-hour and
// weekly limits from stored usage snapshots. Sessions with no snapshot
// coverage are left without an estimate.
func (h *Handlers) AttachLimitShares(sessions []*parser.SessionSummary) {
	if h.store == nil || len(sessions) == 0 {
		return
	}

	var earliest time.Time
	for _, s := range sessions {
		if !s.StartTime.IsZero() && (earliest.IsZero() || s.StartTime.Before(earliest)) {
			earliest = s.StartTime
		}
	}
	if earliest.IsZero() {
		return
	}
	// Load snapshots back to the start of the weekly window the earliest
	// session ran in, and events back far enough that a window open at the
	// first snapshot can be attributed from its start.
	snapSince := earliest.Add(-weeklyWindow)
	eventsFrom := snapSince.Add(-weeklyWindow)

	byID, err := parser.CostEventsSince(h.claudeDir, eventsFrom)
	if err != nil {
		log.Printf("[limitshare] Loading cost events: %v", err)
		return
	}
	var events []limitshare.Event
	for id, evs := range byID {
		for _, e := range evs {
			events = append(events, limitshare.Event{Session: id, Time: e.Time, Cost: e.CostUSD})
		}
	}

	gauges := []struct {
		name   string
		window time.Duration
		set    func(s *parser.SessionSummary, v *float64)
	}{
		{"session", sessionWindow, func(s *parser.SessionSummary, v *float64) { s.SessionLimitPct = v }},
		{"weekly", weeklyWindow, func(s *parser.SessionSummary, v *float64) { s.WeeklyLimitPct = v }},
	}
	for _, g := range gauges {
		stored, err := h.store.GetGaugeSnapshots(g.name, snapSince)
		if err != nil {
			log.Printf("[limitshare] Loading %s snapshots: %v", g.name, err)
			continue
		}
		snaps := make([]limitshare.Snapshot, len(stored))
		for i, st := range stored {
			snaps[i] = limitshare.Snapshot{Time: st.Time, Pct: st.Pct}
			if st.ResetAt != "" {
				if t, err := time.Parse(time.RFC3339, st.ResetAt); err == nil {
					snaps[i].ResetAt = t
				}
			}
		}

		res := limitshare.Attribute(snaps, events, g.window, eventsFrom)
		if res.From.IsZero() {
			continue
		}
		for _, s := range sessions {
			if s.LastActivity.Before(res.From) {
				continue
			}
			v := math.Round(res.BySession[s.ID]*100) / 100
			g.set(s, &v)
		}
	}
}
