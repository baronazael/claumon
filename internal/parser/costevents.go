package parser

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CostEvent is one assistant API response with its estimated USD cost.
type CostEvent struct {
	Time    time.Time
	CostUSD float64
}

// costEventCache memoizes parseSessionFileCostEvents results.
var costEventCache = newFileCache[[]CostEvent]()

// parseSessionFileCostEvents returns one event per assistant API response in
// time order, deduplicated by responseLog.
func parseSessionFileCostEvents(path string) ([]CostEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening session file: %w", err)
	}
	defer f.Close()

	var responses responseLog
	scanner := newSessionScanner(f)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var entry jsonlLine
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if entry.Type != "assistant" || entry.Timestamp.IsZero() {
			continue
		}
		responses.add(&entry)
	}

	events := make([]CostEvent, 0, len(responses.entries))
	for _, r := range responses.entries {
		events = append(events, CostEvent{
			Time:    r.Time,
			CostUSD: costFor(r.Model, r.Input, r.Output, r.CacheRead, r.CacheCreate),
		})
	}
	return events, nil
}

// CostEventsSince returns, keyed by session ID, the cost events at or after
// since. Files last modified before since are skipped without parsing.
func CostEventsSince(claudeDir string, since time.Time) (map[string][]CostEvent, error) {
	files, err := enumerateSessionFiles(claudeDir)
	if err != nil {
		return nil, err
	}
	out := map[string][]CostEvent{}
	for _, f := range files {
		if f.modTime.Before(since) {
			continue
		}
		events, err := costEventCache.get(f, parseSessionFileCostEvents)
		if err != nil {
			continue
		}
		id := strings.TrimSuffix(filepath.Base(f.path), ".jsonl")
		for _, e := range events {
			if !e.Time.Before(since) {
				out[id] = append(out[id], e)
			}
		}
	}
	return out, nil
}
