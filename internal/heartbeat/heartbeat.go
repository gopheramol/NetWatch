// Package heartbeat detects whether the previous process run shut down
// cleanly. A SIGKILL (OOM-kill, `docker kill`, power loss) gives the process
// zero time to run code, so the only way to report "why" it stopped is to
// leave a breadcrumb on disk and check it on the next startup.
package heartbeat

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// State is the on-disk heartbeat record.
type State struct {
	StartedAt time.Time `json:"started_at"`
	LastSeen  time.Time `json:"last_seen"`
	Clean     bool      `json:"clean"`
}

// Tracker persists heartbeat state to a file so it survives process restarts.
type Tracker struct {
	path string
}

// New builds a Tracker backed by the given file path.
func New(path string) *Tracker {
	return &Tracker{path: path}
}

// Previous reads the state left behind by the last run, if any.
func (t *Tracker) Previous() (State, bool) {
	data, err := os.ReadFile(t.path)
	if err != nil {
		return State{}, false
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		return State{}, false
	}
	return s, true
}

// Start records a fresh run as "not yet cleanly stopped".
func (t *Tracker) Start(now time.Time) error {
	return t.write(State{StartedAt: now, LastSeen: now, Clean: false})
}

// Touch refreshes the last-seen timestamp for the current run so an unclean
// stop can be reported with an accurate "last seen" time.
func (t *Tracker) Touch(now time.Time) error {
	s, ok := t.Previous()
	if !ok {
		s = State{StartedAt: now}
	}
	s.LastSeen = now
	s.Clean = false
	return t.write(s)
}

// MarkClean records that the current run is shutting down normally, so the
// next startup won't report it as a crash.
func (t *Tracker) MarkClean(now time.Time) error {
	s, ok := t.Previous()
	if !ok {
		s = State{StartedAt: now}
	}
	s.LastSeen = now
	s.Clean = true
	return t.write(s)
}

func (t *Tracker) write(s State) error {
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(t.path), 0o755); err != nil {
		return err
	}
	// Write-then-rename so a crash mid-write never leaves a corrupt file.
	tmp := t.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, t.path)
}
