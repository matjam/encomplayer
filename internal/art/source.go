package art

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	_ "image/gif"  // Register GIF for embedded covers.
	_ "image/jpeg" // Register JPEG for covers.
	_ "image/png"  // Register PNG for covers.
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/dhowden/tag"
	"golang.org/x/image/draw"
)

// ErrNoArt means neither the file nor its folder has album art.
var ErrNoArt = errors.New("no album art")

var coverNames = []string{"cover", "folder", "front", "album", "albumart", "albumartsmall"}

var coverExts = []string{".jpg", ".jpeg", ".png"}

// Load returns the art embedded in the audio file at path, or a cover image
// in the same folder.
func Load(path string) (image.Image, error) {
	if img, err := embedded(path); err == nil {
		return img, nil
	}
	return folderCover(filepath.Dir(path))
}

func embedded(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, fmt.Errorf("read tags %s: %w", path, err)
	}
	pic := m.Picture()
	if pic == nil || len(pic.Data) == 0 {
		return nil, ErrNoArt
	}
	img, _, err := image.Decode(bytes.NewReader(pic.Data))
	if err != nil {
		return nil, fmt.Errorf("decode embedded art %s: %w", path, err)
	}
	return img, nil
}

func folderCover(dir string) (image.Image, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", dir, err)
	}

	best, bestRank := "", len(coverNames)
	for _, e := range entries {
		name := strings.ToLower(e.Name())
		ext := filepath.Ext(name)
		if e.IsDir() || !slices.Contains(coverExts, ext) {
			continue
		}
		if rank := slices.Index(coverNames, strings.TrimSuffix(name, ext)); rank >= 0 && rank < bestRank {
			best, bestRank = e.Name(), rank
		}
	}
	if best == "" {
		return nil, ErrNoArt
	}

	f, err := os.Open(filepath.Join(dir, best))
	if err != nil {
		return nil, fmt.Errorf("open cover: %w", err)
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("decode cover %s: %w", best, err)
	}
	return img, nil
}

// Fit scales img to fit within w × h pixels, keeping its aspect ratio.
func Fit(img image.Image, w, h int) *image.RGBA {
	b := img.Bounds()
	scale := min(float64(w)/float64(b.Dx()), float64(h)/float64(b.Dy()))
	dw := max(1, int(float64(b.Dx())*scale))
	dh := max(1, int(float64(b.Dy())*scale))

	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst
}
