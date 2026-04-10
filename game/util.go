package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// AntiAlias controls whether vector drawing uses anti-aliasing.
const AntiAlias = true

// drawPolygon draws a regular polygon outline.
func drawPolygon(screen *ebiten.Image, cx, cy, radius float32, sides int, startAngle float64, thickness float32, col color.RGBA) {
	for i := 0; i < sides; i++ {
		a1 := startAngle + float64(i)*2*math.Pi/float64(sides)
		a2 := startAngle + float64(i+1)*2*math.Pi/float64(sides)
		x1 := cx + radius*float32(math.Cos(a1))
		y1 := cy + radius*float32(math.Sin(a1))
		x2 := cx + radius*float32(math.Cos(a2))
		y2 := cy + radius*float32(math.Sin(a2))
		vector.StrokeLine(screen, x1, y1, x2, y2, thickness, col, AntiAlias)
	}
}

// compact removes elements from slice in-place where keep returns false.
func compact[T any](slice []T, keep func(*T) bool) []T {
	n := 0
	for i := range slice {
		if keep(&slice[i]) {
			slice[n] = slice[i]
			n++
		}
	}
	return slice[:n]
}

func lerpColor(a, b color.RGBA, t float32) color.RGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{
		R: uint8(float32(a.R)*(1-t) + float32(b.R)*t),
		G: uint8(float32(a.G)*(1-t) + float32(b.G)*t),
		B: uint8(float32(a.B)*(1-t) + float32(b.B)*t),
		A: uint8(float32(a.A)*(1-t) + float32(b.A)*t),
	}
}
