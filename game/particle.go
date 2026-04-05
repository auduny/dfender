package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Particle struct {
	X, Y     float64
	VX, VY   float64
	Life     int // frames remaining
	MaxLife  int
	Size     float32
	Color    color.RGBA
}

func spawnExplosion(g *Game, x, y float64, col color.RGBA, count int) {
	for i := 0; i < count; i++ {
		angle := rand.Float64() * 2 * math.Pi
		speed := 1.0 + rand.Float64()*4.0
		life := 20 + rand.Intn(30)
		g.Particles = append(g.Particles, Particle{
			X: x, Y: y,
			VX: math.Cos(angle) * speed,
			VY: math.Sin(angle) * speed,
			Life: life, MaxLife: life,
			Size:  2 + float32(rand.Float64()*3),
			Color: col,
		})
	}
}

func spawnDeathExplosion(g *Game, x, y float64) {
	for i := 0; i < 80; i++ {
		angle := rand.Float64() * 2 * math.Pi
		speed := 0.5 + rand.Float64()*6.0
		life := 60 + rand.Intn(90) // 1–2.5 seconds — much longer than normal
		col := ColorPlayer
		// Mix in some white/orange particles for variety.
		if rand.Float64() < 0.3 {
			col = ColorUI
		}
		g.Particles = append(g.Particles, Particle{
			X: x, Y: y,
			VX: math.Cos(angle) * speed,
			VY: math.Sin(angle) * speed,
			Life: life, MaxLife: life,
			Size:  3 + float32(rand.Float64()*4),
			Color: col,
		})
	}
}

func spawnThrustParticles(g *Game, x, y, dirX, dirY float64, col color.RGBA) {
	for i := 0; i < 5; i++ {
		spread := (rand.Float64() - 0.5) * 0.5
		speed := 2.0 + rand.Float64()*2.0
		life := 10 + rand.Intn(10)
		g.Particles = append(g.Particles, Particle{
			X: x, Y: y,
			VX: dirX*speed + spread,
			VY: dirY*speed + spread,
			Life: life, MaxLife: life,
			Size:  1 + float32(rand.Float64()*2),
			Color: col,
		})
	}
}

func updateParticles(g *Game) {
	for i := range g.Particles {
		p := &g.Particles[i]
		p.X += p.VX
		p.Y += p.VY
		p.VX *= 0.97 // drag
		p.VY *= 0.97
		p.Life--
	}
	// Compact.
	n := 0
	for i := range g.Particles {
		if g.Particles[i].Life > 0 {
			g.Particles[n] = g.Particles[i]
			n++
		}
	}
	g.Particles = g.Particles[:n]
}

func drawParticles(screen *ebiten.Image, g *Game, ox, oy float64) {
	for i := range g.Particles {
		p := &g.Particles[i]
		alpha := float32(p.Life) / float32(p.MaxLife)
		col := p.Color
		col.R = uint8(float32(col.R) * alpha)
		col.G = uint8(float32(col.G) * alpha)
		col.B = uint8(float32(col.B) * alpha)
		col.A = uint8(float32(col.A) * alpha)
		cx := float32(p.X + ox)
		cy := float32(p.Y + oy)
		vector.DrawFilledCircle(screen, cx, cy, p.Size*alpha, col, false)
	}
}
