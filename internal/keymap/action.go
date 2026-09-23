package keymap

import (
	"fmt"
	"strings"
)

// Action is a named command, optionally with an argument, as in rmpc's
// SwitchToTab("Queue").
type Action struct {
	Name string
	Arg  string
}

// String renders the action in rmpc's config syntax.
func (a Action) String() string {
	if a.Arg == "" {
		return a.Name
	}
	return fmt.Sprintf("%s(%q)", a.Name, a.Arg)
}

// ParseAction parses "Quit" or `SwitchToTab("Queue")`.
func ParseAction(s string) (Action, error) {
	s = strings.TrimSpace(s)
	open := strings.IndexByte(s, '(')
	if open < 0 {
		if s == "" {
			return Action{}, fmt.Errorf("empty action")
		}
		return Action{Name: s}, nil
	}
	if !strings.HasSuffix(s, ")") {
		return Action{}, fmt.Errorf("action %q: missing closing parenthesis", s)
	}
	arg := strings.TrimSpace(s[open+1 : len(s)-1])
	arg = strings.Trim(arg, `"`)
	return Action{Name: s[:open], Arg: arg}, nil
}

// Action names, as rmpc spells them.
const (
	Quit                = "Quit"
	ShowHelp            = "ShowHelp"
	CommandMode         = "CommandMode"
	ShowCurrentSongInfo = "ShowCurrentSongInfo"
	ToggleRepeat        = "ToggleRepeat"
	ToggleRandom        = "ToggleRandom"
	ToggleConsume       = "ToggleConsume"
	ToggleSingle        = "ToggleSingle"
	TogglePause         = "TogglePause"
	Stop                = "Stop"
	NextTrack           = "NextTrack"
	PreviousTrack       = "PreviousTrack"
	SeekForward         = "SeekForward"
	SeekBack            = "SeekBack"
	VolumeUp            = "VolumeUp"
	VolumeDown          = "VolumeDown"
	NextTab             = "NextTab"
	PreviousTab         = "PreviousTab"
	SwitchToTab         = "SwitchToTab"
	Update              = "Update"
	Rescan              = "Rescan"
	AddRandom           = "AddRandom"

	Close           = "Close"
	Confirm         = "Confirm"
	Up              = "Up"
	Down            = "Down"
	Left            = "Left"
	Right           = "Right"
	PaneUp          = "PaneUp"
	PaneDown        = "PaneDown"
	PaneLeft        = "PaneLeft"
	PaneRight       = "PaneRight"
	MoveUp          = "MoveUp"
	MoveDown        = "MoveDown"
	UpHalf          = "UpHalf"
	DownHalf        = "DownHalf"
	PageUp          = "PageUp"
	PageDown        = "PageDown"
	Top             = "Top"
	Bottom          = "Bottom"
	Select          = "Select"
	InvertSelection = "InvertSelection"
	EnterSearch     = "EnterSearch"
	NextResult      = "NextResult"
	PreviousResult  = "PreviousResult"
	Add             = "Add"
	AddAll          = "AddAll"
	Delete          = "Delete"
	Rename          = "Rename"
	FocusInput      = "FocusInput"
	ShowInfo        = "ShowInfo"
	Save            = "Save"
	SaveAll         = "SaveAll"

	DeleteAll     = "DeleteAll"
	Play          = "Play"
	JumpToCurrent = "JumpToCurrent"
	Shuffle       = "Shuffle"

	// EncomPlayer additions, not in rmpc.
	ShuffleAll  = "ShuffleAll"
	ShufflePlay = "ShufflePlay"
)
