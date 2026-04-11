package game

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// generateWindowIcon renders "dF" + player hexagon on a dark background.
func (g *Game) generateWindowIcon() {
	const size = 128

	scene := ebiten.NewImage(size, size)
	defer scene.Deallocate()

	scene.Fill(ColorBackground)

	cx := float32(size) / 2

	// "dF" title text.
	face := &text.GoTextFace{Source: FontTitle.Source, Size: 52}
	w, _ := text.Measure("dF", face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(size)/2-w/2, float64(size)*0.08)
	op.ColorScale.ScaleWithColor(ColorBorder)
	text.Draw(scene, "dF", face, op)

	// Player hexagon ship.
	shipY := float32(size) * 0.65
	vector.StrokeCircle(scene, cx, shipY, 25, 4, ColorBorderDim, true)
	drawPolygon(scene, cx, shipY, 22, 6, -pi32/2, 3, ColorPlayer)
	vector.DrawFilledCircle(scene, cx, shipY, 4, ColorPlayer, true)

	// Read pixels into a standard image.
	pixels := make([]byte, 4*size*size)
	scene.ReadPixels(pixels)

	img := image.NewRGBA(image.Rect(0, 0, size, size))
	copy(img.Pix, pixels)

	ebiten.SetWindowIcon([]image.Image{img})
	setMacOSDockIcon(pixels, size, size)
}
