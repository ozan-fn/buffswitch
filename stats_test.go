package main

import (
	"path/filepath"
	"testing"
	"time"
)

func TestLoadStatsMissingFile(t *testing.T) {
	st := LoadStats(filepath.Join(t.TempDir(), "nope.json"))
	if got := st.Get("a@x.com"); got.Count != 0 {
		t.Fatalf("expected zero usage, got %+v", got)
	}
}

func TestStatsSessionRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "buffswitch-stats.json")
	st := LoadStats(path)

	st.Begin("a@x.com")
	st.Begin("a@x.com") // second session, same account
	// pretend the open session started 2h ago
	st.Usage["a@x.com"].StartedAt = time.Now().Add(-2 * time.Hour)
	st.Begin("b@x.com")

	// data survives a reload (bs restarts between sessions)
	st2 := LoadStats(path)
	if got := st2.Get("a@x.com").Count; got != 2 {
		t.Fatalf("a count = %d, want 2", got)
	}
	if st2.Get("a@x.com").StartedAt.IsZero() {
		t.Fatal("a session should still be open after reload")
	}

	st2.End("a@x.com")
	u := st2.Get("a@x.com")
	if !u.StartedAt.IsZero() {
		t.Fatal("a session should be closed after End")
	}
	if u.TotalSeconds < 7199 {
		t.Fatalf("a totalSeconds = %d, want ~7200", u.TotalSeconds)
	}
	if u.Summary() == "" {
		t.Fatal("empty summary for a used account")
	}

	// End on an account without an open session is a no-op
	st2.End("nobody@x.com")
	if st2.Get("nobody@x.com").Count != 0 {
		t.Fatal("End invented usage for an unknown account")
	}
}
