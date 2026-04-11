package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// AntiAlias controls whether vector drawing uses anti-aliasing.
const AntiAlias = true

// drawPolygon draws a regular polygon outline.
func drawPolygon(screen *ebiten.Image, cx, cy, radius float32, sides int, startAngle float32, thickness float32, col color.RGBA) {
	for i := 0; i < sides; i++ {
		a1 := startAngle + float32(i)*2*pi32/float32(sides)
		a2 := startAngle + float32(i+1)*2*pi32/float32(sides)
		x1 := cx + radius*cos32(a1)
		y1 := cy + radius*sin32(a1)
		x2 := cx + radius*cos32(a2)
		y2 := cy + radius*sin32(a2)
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
