package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/matjam/encomplayer/internal/domain"
	"github.com/matjam/encomplayer/internal/fsutil"
)

// State is what EncomPlayer restores on the next launch.
type State struct {
	Queue   []string     `json:"queue"`
	Current int          `json:"current"`
	Modes   domain.Modes `json:"modes"`
	Volume  int          `json:"volume"`
	Tab     string       `json:"tab"`
	Layout  Layout       `json:"layout"`

	// PositionSeconds is how far into the current track playback had got,
	// so the next launch can resume there.
	PositionSeconds float64 `json:"position_seconds"`

	// VisualizerFull is true when the visualiser filled the screen.
	VisualizerFull bool `json:"visualizer_full"`
}

// Layout holds the pane sizes the user set by dragging dividers.
type Layout struct {
	// FooterRows is the height of the signal strip, borders included.
	FooterRows int `json:"footer_rows"`

	// ArtPercent is the album art panel's share of the queue tab width.
	ArtPercent int `json:"art_percent"`

	// ParentPercent and PreviewPercent are the outer browser columns'
	// shares of the width; the current column gets the rest.
	ParentPercent  int `json:"parent_percent"`
	PreviewPercent int `json:"preview_percent"`
}

// DefaultLayout is the layout before any divider is dragged.
func DefaultLayout() Layout {
	return Layout{FooterRows: 4, ArtPercent: 35, ParentPercent: 25, PreviewPercent: 30}
}

// DefaultState is used on first launch.
func DefaultState() State {
	return State{Current: -1, Volume: 70, Layout: DefaultLayout()}
}

// LoadState reads saved state. A missing or unreadable file yields defaults,
// because losing the previous queue should never stop the player starting.
func LoadState(path string) State {
	s := DefaultState()
	data, err := os.ReadFile(path)
	if err != nil {
		return s
	}
	if json.Unmarshal(data, &s) != nil {
		return DefaultState()
	}
	if s.Layout == (Layout{}) {
		s.Layout = DefaultLayout()
	}
	return s
}

// SaveState writes state atomically.
func SaveState(path string, s State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	return fsutil.WriteFileAtomic(path, data)
}
