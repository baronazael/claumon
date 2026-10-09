// Package limitshare estimates how much of a rate-limit window (the 5-hour
// session limit or the 7-day weekly limit) each Claude Code session used.
//
// The usage API only reports one aggregate utilization percentage per
// window, so per-session shares are an estimate: every increase observed
// between two usage snapshots is split across the local assistant messages
// sent since the previous increase, proportionally to their API-equivalent
// cost. Increases with no local messages behind them (claude.ai, another
// machine) are reported as unattributed.
package limitshare

import (
	"sort"
	"time"
)

// Snapshot is one poll of a single gauge.
type Snapshot struct {
	Time    time.Time
	Pct     float64   // utilization, 0-100
	ResetAt time.Time // zero when no window is active
}

// Event is one assistant message with its estimated cost.
type Event struct {
	Session string
	Time    time.Time
	Cost    float64
}

// Result holds the estimated percentage points of the gauge used by each
// session.
type Result struct {
	BySession map[string]float64
	// Unattributed is usage that rose while no local session was active.
	Unattributed float64
	// From is the earliest time attribution is reliable from. Sessions with
	// no activity after it have no estimate. Zero if nothing was attributed.
	From time.Time
}

const (
	// resetTolerance absorbs reset_at drift between polls of the same window
	// that minute normalization at write time does not catch.
	resetTolerance = 2 * time.Minute
	// windowStartSlack widens a window's derived start, since reset_at is
	// rounded to the minute.
	windowStartSlack = time.Minute
)

// Attribute splits the gauge increases in snaps across events. window is the
// gauge's length (5h or 7d). eventsFrom is how far back events were loaded:
// a window that started before it can't be attributed from its start, so
// attribution only begins at that window's next observed increase.
//
// Snapshots must be in ascending time order. events may be in any order.
func Attribute(snaps []Snapshot, events []Event, window time.Duration, eventsFrom time.Time) Result {
	res := Result{BySession: map[string]float64{}}
	if len(snaps) == 0 {
		return res
	}

	sorted := make([]Event, len(events))
	copy(sorted, events)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Time.Before(sorted[j].Time) })

	// distribute splits pct across events in (from, to].
	distribute := func(pct float64, from, to time.Time) {
		lo := sort.Search(len(sorted), func(i int) bool { return sorted[i].Time.After(from) })
		hi := sort.Search(len(sorted), func(i int) bool { return sorted[i].Time.After(to) })
		var total float64
		for _, e := range sorted[lo:hi] {
			total += e.Cost
		}
		if total <= 0 {
			res.Unattributed += pct
			return
		}
		for _, e := range sorted[lo:hi] {
			res.BySession[e.Session] += pct * e.Cost / total
		}
	}

	// segStart is the exclusive start of the usage not yet reflected in an
	// attributed increase. known is false while that start is unknown.
	var segStart time.Time
	known := false
	begin := func(t time.Time) {
		segStart = t
		known = true
		if res.From.IsZero() {
			res.From = t
		}
	}

	first := snaps[0]
	if !first.ResetAt.IsZero() {
		if start := first.ResetAt.Add(-window - windowStartSlack); !start.Before(eventsFrom) {
			begin(start)
			if first.Pct > 0 {
				distribute(first.Pct, segStart, first.Time)
				segStart = first.Time
			}
		}
	}

	for i := 1; i < len(snaps); i++ {
		prev, cur := snaps[i-1], snaps[i]

		if cur.ResetAt.IsZero() {
			// No active window: anything pending was already counted or lost
			// to rounding. The next window starts fresh.
			known = false
			continue
		}

		if isNewWindow(prev, cur, window) {
			start := cur.ResetAt.Add(-window - windowStartSlack)
			if start.Before(prev.Time) {
				start = prev.Time
			}
			begin(start)
			if cur.Pct > 0 {
				distribute(cur.Pct, segStart, cur.Time)
				segStart = cur.Time
			}
			continue
		}

		delta := cur.Pct - prev.Pct
		if delta <= 0 {
			continue
		}
		if !known {
			// The usage behind this increase may predate the loaded events.
			begin(cur.Time)
			continue
		}
		distribute(delta, segStart, cur.Time)
		segStart = cur.Time
	}

	return res
}

// isNewWindow reports whether cur belongs to a different window than prev.
func isNewWindow(prev, cur Snapshot, window time.Duration) bool {
	if prev.ResetAt.IsZero() {
		return true
	}
	diff := cur.ResetAt.Sub(prev.ResetAt)
	if diff < 0 {
		diff = -diff
	}
	if diff <= resetTolerance {
		return false
	}
	// Windows don't overlap: one active at prev.Time can't be followed by one
	// that started before it. Treat that as reset_at drift.
	start := cur.ResetAt.Add(-window)
	return !start.Before(prev.Time.Add(-windowStartSlack))
}
