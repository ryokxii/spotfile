// Package activity keeps the Dashboard's short local history: recent
// searches, recently opened files and the last file the index updated. It is
// stored only on this computer, in ~/.spotfile/activity.json.
package activity

import (
	"strings"
	"time"

	"spotfile/engine/jsonfile"
)

// MaxEntries caps each history list.
const MaxEntries = 50

// Search is a query that returned results.
type Search struct {
	Query string    `json:"query"`
	At    time.Time `json:"at"`
}

// Opened is a file opened in a viewer.
type Opened struct {
	Path string    `json:"path"`
	At   time.Time `json:"at"`
}

// Update is the last file the index (re)indexed.
type Update struct {
	Path string    `json:"path"`
	At   time.Time `json:"at"`
}

// Log is the whole history. Every function in this package returns a new Log
// and leaves its argument untouched.
type Log struct {
	Searches   []Search `json:"searches"`   // newest first
	Opened     []Opened `json:"opened"`     // newest first, one entry per path
	LastUpdate *Update  `json:"lastUpdate"` // nil until the index first changes
}

// Empty is a log with no history.
func Empty() Log {
	return Log{Searches: []Search{}, Opened: []Opened{}}
}

func (l Log) clone() Log {
	out := Log{Searches: append([]Search{}, l.Searches...), Opened: append([]Opened{}, l.Opened...)}
	if l.LastUpdate != nil {
		u := *l.LastUpdate
		out.LastUpdate = &u
	}
	return out
}

// WithSearch puts query at the top of the searches. A blank query is ignored;
// a repeat (ignoring case and surrounding spaces) moves to the top.
func WithSearch(l Log, query string, now time.Time) Log {
	out := l.clone()
	query = strings.TrimSpace(query)
	if query == "" {
		return out
	}
	next := []Search{{Query: query, At: now}}
	for _, s := range l.Searches {
		if len(next) == MaxEntries {
			break
		}
		if !strings.EqualFold(s.Query, query) {
			next = append(next, s)
		}
	}
	out.Searches = next
	return out
}

// WithOpened puts path at the top of the opened files.
func WithOpened(l Log, path string, now time.Time) Log {
	out := l.clone()
	next := []Opened{{Path: path, At: now}}
	for _, o := range l.Opened {
		if len(next) == MaxEntries {
			break
		}
		if o.Path != path {
			next = append(next, o)
		}
	}
	out.Opened = next
	return out
}

// WithoutOpened forgets path from the opened files.
func WithoutOpened(l Log, path string) Log {
	out := l.clone()
	next := make([]Opened, 0, len(l.Opened))
	for _, o := range l.Opened {
		if o.Path != path {
			next = append(next, o)
		}
	}
	out.Opened = next
	return out
}

// WithUpdate records path as the last file the index updated.
func WithUpdate(l Log, path string, now time.Time) Log {
	out := l.clone()
	out.LastUpdate = &Update{Path: path, At: now}
	return out
}

// Cleared forgets searches and opened files. The last update describes the
// library rather than the user, so it is kept.
func Cleared(l Log) Log {
	out := l.clone()
	out.Searches = []Search{}
	out.Opened = []Opened{}
	return out
}

// LastOpened maps each opened file to when it was last opened.
func LastOpened(l Log) map[string]time.Time {
	out := make(map[string]time.Time, len(l.Opened))
	for _, o := range l.Opened {
		out[o.Path] = o.At
	}
	return out
}

// Load reads the log. A missing file gives an empty log; an unusable one is
// set aside and an empty log is returned with the error.
func Load(path string) (Log, error) {
	l := Empty()
	if err := jsonfile.Load(path, &l); err != nil {
		return Empty(), err
	}
	if l.Searches == nil {
		l.Searches = []Search{}
	}
	if l.Opened == nil {
		l.Opened = []Opened{}
	}
	return l, nil
}

// Save writes the log atomically.
func Save(path string, l Log) error {
	return jsonfile.Save(path, l)
}
