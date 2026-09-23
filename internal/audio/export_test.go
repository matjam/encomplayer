package audio

// MemorySource exposes the player's in-memory source to external tests.
func MemorySource(path string) (Source, error) {
	f, err := loadFile(path)
	if err != nil {
		return Source{}, err
	}
	return Source{Path: path, Open: f.open}, nil
}
