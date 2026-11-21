// genicon renders the pet into winres/icon.png (app icon) and, optionally,
// a preview sheet of every animation state: go run ./cmd/genicon [preview.png]
package main

import (
	"image"
	"image/color"
	"image/png"
	"os"

	"pomopet/sprite"
)

func fillOn(img *image.RGBA) sprite.FillFunc {
	return func(x, y, w, h int, c uint32) {
		col := color.RGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255}
		for yy := y; yy < y+h; yy++ {
			for xx := x; xx < x+w; xx++ {
				if image.Pt(xx, yy).In(img.Bounds()) {
					img.SetRGBA(xx, yy, col)
				}
			}
		}
	}
}

func save(img image.Image, path string) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

func main() {
	// 256x256 icon: 16 grid cells * 15px = 240 wide, centred.
	icon := image.NewRGBA(image.Rect(0, 0, 256, 256))
	sprite.Draw(fillOn(icon), 8, 68, 15, sprite.Idle, 0.2)
	save(icon, "winres/icon.png")

	if len(os.Args) > 1 {
		states := []sprite.State{sprite.Idle, sprite.Focus, sprite.Break, sprite.Paused, sprite.Alert}
		p := 8.0
		cw, ch := 180, 130
		sheet := image.NewRGBA(image.Rect(0, 0, cw*len(states), ch))
		for i := range sheet.Pix {
			sheet.Pix[i] = 0xF4
		}
		for i, st := range states {
			sprite.Draw(fillOn(sheet), i*cw+(cw-int(sprite.BodyW*p))/2, 50, p, st, 0.5)
		}
		save(sheet, os.Args[1])
	}
}
