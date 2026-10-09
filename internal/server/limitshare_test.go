package server

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fabioconcina/claumon/internal/parser"
	"github.com/fabioconcina/claumon/internal/pricing"
)

func TestAttachLimitShares(t *testing.T) {
	parser.SetPricingTable(pricing.Load(nil))
	srv, st := setupTestServer(t)
	claudeDir := srv.Handlers.claudeDir

	now := time.Now().UTC()
	proj := filepath.Join(claudeDir, "projects", "-tmp-proj")
	if err := os.MkdirAll(proj, 0755); err != nil {
		t.Fatal(err)
	}
	// Session "big" costs 3x session "small", both inside the current window.
	line := func(id string, ts time.Time, in int) string {
		return fmt.Sprintf(`{"type":"assistant","timestamp":%q,"message":{"id":%q,"model":"claude-sonnet-4-6","usage":{"input_tokens":%d,"output_tokens":0}}}`+"\n",
			ts.Format(time.RFC3339), id, in)
	}
	write := func(name, content string) {
		if err := os.WriteFile(filepath.Join(proj, name+".jsonl"), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	write("big", line("m1", now.Add(-30*time.Minute), 3000))
	write("small", line("m2", now.Add(-10*time.Minute), 1000))

	// The 5h window opened 1h ago; the weekly window 1 day ago.
	sessionReset := now.Add(4 * time.Hour).Format(time.RFC3339)
	weeklyReset := now.Add(6 * 24 * time.Hour).Format(time.RFC3339)
	if err := st.SaveUsageSnapshot(8, 2, sessionReset, weeklyReset, json.RawMessage(`{}`)); err != nil {
		t.Fatal(err)
	}

	sessions, err := parser.DiscoverSessions(claudeDir)
	if err != nil {
		t.Fatal(err)
	}
	srv.Handlers.AttachLimitShares(sessions)

	want := map[string][2]float64{"big": {6, 1.5}, "small": {2, 0.5}}
	for _, s := range sessions {
		w, ok := want[s.ID]
		if !ok {
			continue
		}
		if s.SessionLimitPct == nil || *s.SessionLimitPct != w[0] {
			t.Errorf("%s SessionLimitPct = %v, want %v", s.ID, s.SessionLimitPct, w[0])
		}
		if s.WeeklyLimitPct == nil || *s.WeeklyLimitPct != w[1] {
			t.Errorf("%s WeeklyLimitPct = %v, want %v", s.ID, s.WeeklyLimitPct, w[1])
		}
	}
}

func TestAttachLimitSharesWithoutSnapshots(t *testing.T) {
	srv, _ := setupTestServer(t)
	s := &parser.SessionSummary{ID: "x", StartTime: time.Now(), LastActivity: time.Now()}
	srv.Handlers.AttachLimitShares([]*parser.SessionSummary{s})
	if s.SessionLimitPct != nil || s.WeeklyLimitPct != nil {
		t.Errorf("expected no estimate without snapshots, got %v / %v", s.SessionLimitPct, s.WeeklyLimitPct)
	}
}
