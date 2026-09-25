package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// statsPath is bs's usage data, next to credentials.json (follows -creds) —
// kept separate so credentials.json stays untouched.
func statsPath(credsPath string) string {
	return filepath.Join(filepath.Dir(credsPath), "buffswitch-stats.json")
}

// Usage is the tracked usage of one account.
type Usage struct {
	TotalSeconds int64     `json:"totalSeconds"`
	Count        int       `json:"count"`
	LastUsed     time.Time `json:"lastUsed"`
	StartedAt    time.Time `json:"startedAt,omitempty"` // set while a session is open
}

// Stats wraps the usage file, keyed by account email.
type Stats struct {
	Path  string
	Usage map[string]*Usage
}

// LoadStats reads the usage file (empty stats when missing or broken).
func LoadStats(path string) *Stats {
	st := &Stats{Path: path, Usage: map[string]*Usage{}}
	data, err := os.ReadFile(path)
	if err != nil {
		return st
	}
	_ = json.Unmarshal(data, &st.Usage)
	return st
}

// Get returns the usage entry for an email (zero value when unknown).
func (st *Stats) Get(email string) Usage {
	if st == nil {
		return Usage{}
	}
	if u, ok := st.Usage[email]; ok && u != nil {
		return *u
	}
	return Usage{}
}

// Begin opens a session for the account: count and lastUsed tick now.
// Nil-safe so callers never need a guard.
func (st *Stats) Begin(email string) {
	if st == nil || email == "" {
		return
	}
	u, ok := st.Usage[email]
	if !ok || u == nil {
		u = &Usage{}
		st.Usage[email] = u
	}
	now := time.Now()
	u.Count++
	u.LastUsed = now
	u.StartedAt = now
	st.save()
}

// End closes the open session, accumulating its duration. A no-op when the
// session was never opened (or already closed). A crashed run loses that
// session's duration — Begin restarts the clock.
func (st *Stats) End(email string) {
	if st == nil {
		return
	}
	u, ok := st.Usage[email]
	if !ok || u == nil || u.StartedAt.IsZero() {
		return
	}
	u.TotalSeconds += int64(time.Since(u.StartedAt).Seconds())
	u.StartedAt = time.Time{}
	st.save()
}

func (st *Stats) save() {
	data, err := json.MarshalIndent(st.Usage, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(st.Path, append(data, '\n'), 0o600)
}

// Summary is the one-line list suffix, e.g. "12x · 3j · 2j lalu".
func (u Usage) Summary() string {
	if u.Count == 0 {
		return ""
	}
	s := fmt.Sprintf("%dx · %s", u.Count, fmtDur(u.TotalSeconds))
	if rel := relTime(u.LastUsed); rel != "belum pernah" {
		s += " · " + rel
	}
	return s
}

// fmtDur renders seconds as a compact Indonesian duration.
func fmtDur(secs int64) string {
	switch {
	case secs < 60:
		return fmt.Sprintf("%ds", secs)
	case secs < 3600:
		return fmt.Sprintf("%dm", secs/60)
	case secs < 86400:
		return fmt.Sprintf("%dj", secs/3600)
	default:
		return fmt.Sprintf("%dhr", secs/86400)
	}
}

// relTime renders a timestamp relative to now.
func relTime(t time.Time) string {
	if t.IsZero() {
		return "belum pernah"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "baru saja"
	case d < time.Hour:
		return fmt.Sprintf("%dm lalu", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dj lalu", int(d.Hours()))
	default:
		return fmt.Sprintf("%dhr lalu", int(d.Hours()/24))
	}
}
