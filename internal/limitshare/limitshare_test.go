package limitshare

import (
	"math"
	"testing"
	"time"
)

var t0 = time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)

func at(min int) time.Time { return t0.Add(time.Duration(min) * time.Minute) }

const window = 5 * time.Hour

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

func TestAttribute(t *testing.T) {
	// A window that started at t0 and resets at t0+5h.
	reset := t0.Add(window)
	nextReset := reset.Add(window)

	tests := []struct {
		name         string
		snaps        []Snapshot
		events       []Event
		eventsFrom   time.Time
		want         map[string]float64
		unattributed float64
		from         time.Time
	}{
		{
			name: "initial window split by cost",
			snaps: []Snapshot{
				{Time: at(30), Pct: 10, ResetAt: reset},
			},
			events: []Event{
				{Session: "a", Time: at(5), Cost: 3},
				{Session: "b", Time: at(20), Cost: 1},
			},
			eventsFrom: t0.Add(-time.Hour),
			want:       map[string]float64{"a": 7.5, "b": 2.5},
			from:       t0.Add(-windowStartSlack),
		},
		{
			name: "zero-delta polls accumulate into the next increase",
			snaps: []Snapshot{
				{Time: at(10), Pct: 0, ResetAt: reset},
				{Time: at(12), Pct: 0, ResetAt: reset},
				{Time: at(14), Pct: 1, ResetAt: reset},
			},
			events: []Event{
				{Session: "a", Time: at(11), Cost: 1},
				{Session: "b", Time: at(13), Cost: 1},
			},
			eventsFrom: t0.Add(-time.Hour),
			want:       map[string]float64{"a": 0.5, "b": 0.5},
			from:       t0.Add(-windowStartSlack),
		},
		{
			name: "increase with no local events is unattributed",
			snaps: []Snapshot{
				{Time: at(10), Pct: 2, ResetAt: reset},
				{Time: at(12), Pct: 5, ResetAt: reset},
			},
			events: []Event{
				{Session: "a", Time: at(5), Cost: 1},
			},
			eventsFrom:   t0.Add(-time.Hour),
			want:         map[string]float64{"a": 2},
			unattributed: 3,
			from:         t0.Add(-windowStartSlack),
		},
		{
			name: "window start before event coverage discards the first partial segment",
			snaps: []Snapshot{
				{Time: at(60), Pct: 20, ResetAt: reset},
				{Time: at(62), Pct: 20, ResetAt: reset},
				{Time: at(64), Pct: 21, ResetAt: reset},
				{Time: at(66), Pct: 23, ResetAt: reset},
			},
			events: []Event{
				{Session: "old", Time: at(61), Cost: 5},
				{Session: "a", Time: at(65), Cost: 1},
			},
			eventsFrom: at(30),
			want:       map[string]float64{"a": 2},
			from:       at(64),
		},
		{
			name: "rollover attributes the new window only to events after its start",
			snaps: []Snapshot{
				{Time: at(290), Pct: 50, ResetAt: reset},
				{Time: at(310), Pct: 4, ResetAt: nextReset},
			},
			events: []Event{
				{Session: "a", Time: at(250), Cost: 50},
				{Session: "old-tail", Time: at(295), Cost: 9},
				{Session: "b", Time: at(305), Cost: 1},
			},
			eventsFrom: t0.Add(-time.Hour),
			want:       map[string]float64{"a": 50, "b": 4},
			from:       t0.Add(-windowStartSlack),
		},
		{
			name: "reset drift within tolerance stays in the same window",
			snaps: []Snapshot{
				{Time: at(10), Pct: 2, ResetAt: reset},
				{Time: at(12), Pct: 3, ResetAt: reset.Add(time.Minute)},
			},
			events: []Event{
				{Session: "a", Time: at(5), Cost: 1},
				{Session: "b", Time: at(11), Cost: 1},
			},
			eventsFrom: t0.Add(-time.Hour),
			want:       map[string]float64{"a": 2, "b": 1},
			from:       t0.Add(-windowStartSlack),
		},
		{
			name: "window ends, then a new one starts after an idle gap",
			snaps: []Snapshot{
				{Time: at(10), Pct: 2, ResetAt: reset},
				{Time: at(400), Pct: 0},
				{Time: at(500), Pct: 3, ResetAt: at(480).Add(window)},
			},
			events: []Event{
				{Session: "a", Time: at(5), Cost: 1},
				{Session: "idle-gap", Time: at(450), Cost: 1},
				{Session: "b", Time: at(490), Cost: 1},
			},
			eventsFrom: t0.Add(-time.Hour),
			want:       map[string]float64{"a": 2, "b": 3},
			from:       t0.Add(-windowStartSlack),
		},
		{
			name:       "no snapshots",
			eventsFrom: t0,
			want:       map[string]float64{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Attribute(tt.snaps, tt.events, window, tt.eventsFrom)
			if len(got.BySession) != len(tt.want) {
				t.Fatalf("BySession = %v, want %v", got.BySession, tt.want)
			}
			for s, w := range tt.want {
				if !approx(got.BySession[s], w) {
					t.Errorf("BySession[%q] = %v, want %v (all: %v)", s, got.BySession[s], w, got.BySession)
				}
			}
			if !approx(got.Unattributed, tt.unattributed) {
				t.Errorf("Unattributed = %v, want %v", got.Unattributed, tt.unattributed)
			}
			if !got.From.Equal(tt.from) {
				t.Errorf("From = %v, want %v", got.From, tt.from)
			}
		})
	}
}

func TestAttributeIgnoresEventOrder(t *testing.T) {
	reset := t0.Add(window)
	snaps := []Snapshot{{Time: at(30), Pct: 4, ResetAt: reset}}
	events := []Event{
		{Session: "b", Time: at(20), Cost: 1},
		{Session: "a", Time: at(5), Cost: 1},
	}
	got := Attribute(snaps, events, window, t0.Add(-time.Hour))
	if !approx(got.BySession["a"], 2) || !approx(got.BySession["b"], 2) {
		t.Errorf("BySession = %v, want a=2 b=2", got.BySession)
	}
	if events[0].Session != "b" {
		t.Error("Attribute must not reorder the caller's slice")
	}
}
