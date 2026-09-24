package art

import (
	"image"
	"testing"
)

func BenchmarkSixelRender(b *testing.B) {
	img := testImage()
	for _, box := range []Box{{Cols: 40, Rows: 20, Cell: image.Pt(10, 21)}, {Cols: 100, Rows: 50, Cell: image.Pt(10, 21)}} {
		b.Run(image.Pt(box.Cols, box.Rows).String(), func(b *testing.B) {
			for b.Loop() {
				if _, err := (SixelRenderer{}).Render(img, box); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
