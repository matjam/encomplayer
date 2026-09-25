package domain

// Playlist is a named, ordered list of file paths.
type Playlist struct {
	Name  string
	Paths []string
}

// Modes are the playback modes rmpc exposes as toggles.
type Modes struct {
	// Repeat wraps the queue at the end, or repeats one track with Single.
	Repeat bool `json:"repeat"`

	// Random picks the next track at random.
	Random bool `json:"random"`

	// Single stops after the current track, or repeats it with Repeat.
	Single bool `json:"single"`

	// Consume removes each track from the queue once it has played.
	Consume bool `json:"consume"`

	// Careful keeps tracks by the same artist apart when shuffling or
	// picking at random. An EncomPlayer addition.
	Careful bool `json:"careful"`
}
