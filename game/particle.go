package game

import (
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type Particle struct {
	X, Y     float32
	VX, VY   float32
	Life     int // frames remaining
	MaxLife  int
	Size     float32
	Color    color.RGBA
}

func spawnExplosion(g *Game, x, y float32, col color.RGBA, count int) {
	for i := 0; i < count; i++ {
		angle := rand.Float32() * 2 * pi32
		speed := 1.0 + rand.Float32()*4.0
		life := 40 + rand.Intn(50)
		g.Particles = append(g.Particles, Particle{
			X: x, Y: y,
			VX: cos32(angle) * speed,
			VY: sin32(angle) * speed,
			Life: life, MaxLife: life,
			Size:  3 + rand.Float32()*4,
			Color: col,
		})
	}
}

func spawnDeathExplosion(g *Game, x, y float32) {
	for i := 0; i < 80; i++ {
		angle := rand.Float32() * 2 * pi32
		speed := 0.5 + rand.Float32()*6.0
		life := 60 + rand.Intn(90) // 1–2.5 seconds — much longer than normal
		col := ColorPlayer
		// Mix in some white/orange particles for variety.
		if rand.Float64() < 0.3 {
			col = ColorUI
		}
		g.Particles = append(g.Particles, Particle{
			X: x, Y: y,
			VX: cos32(angle) * speed,
			VY: sin32(angle) * speed,
			Life: life, MaxLife: life,
			Size:  3 + rand.Float32()*4,
			Color: col,
		})
	}
}

func spawnThrustParticles(g *Game, x, y, dirX, dirY float32, col color.RGBA) {
	for i := 0; i < 5; i++ {
		spread := (rand.Float32() - 0.5) * 0.5
		speed := 2.0 + rand.Float32()*2.0
		life := 10 + rand.Intn(10)
		g.Particles = append(g.Particles, Particle{
			X: x, Y: y,
			VX: dirX*speed + spread,
			VY: dirY*speed + spread,
			Life: life, MaxLife: life,
			Size:  1 + rand.Float32()*2,
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
	g.Particles = compact(g.Particles, func(p *Particle) bool { return p.Life > 0 })
}

func clampByte(v float32) uint8 {
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func drawParticles(screen *ebiten.Image, g *Game, ox, oy float32) {
	for i := range g.Particles {
		p := &g.Particles[i]
		t := float32(p.Life) / float32(p.MaxLife)
		// Brightness boost: particles start at 1.5x brightness and fade to 0.
		var brightness float32
		if t > 0.7 {
			brightness = 1.5 // hot start
		} else {
			brightness = 1.5 * (t / 0.7) // fade out over remaining life
		}
		col := p.Color
		col.R = clampByte(float32(col.R) * brightness)
		col.G = clampByte(float32(col.G) * brightness)
		col.B = clampByte(float32(col.B) * brightness)
		col.A = uint8(float32(col.A) * t)
		cx := p.X + ox
		cy := p.Y + oy
		vector.DrawFilledCircle(screen, cx, cy, p.Size*t, col, AntiAlias)
	}
}
