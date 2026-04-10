package game

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	MineRadius      = 10.0
	MineBlastRadius = 160.0
	MineMaxCount    = 9

	mineBlastRadSq = MineBlastRadius * MineBlastRadius
	mineEnemyRadSq = (MineRadius + EnemyRadius) * (MineRadius + EnemyRadius)
)

var (
	colorMineSpike = color.RGBA{0xFF, 0xCC, 0x33, 0xFF} // gold spikes
	colorMinePulse = color.RGBA{0xFF, 0x33, 0x00, 0xAA} // red pulse
	colorMineBlast = color.RGBA{0xFF, 0xAA, 0x00, 0xFF} // orange blast
)

// Precomputed spike directions (8 evenly spaced angles).
var spikeDirs [8][2]float32

func init() {
	for i := 0; i < 8; i++ {
		a := float64(i) * math.Pi / 4
		spikeDirs[i] = [2]float32{float32(math.Cos(a)), float32(math.Sin(a))}
	}
}

type Mine struct {
	X, Y  float64
	Age   int
	Alive bool
}

func updateMines(g *Game) {
	for i := range g.Mines {
		m := &g.Mines[i]
		if !m.Alive {
			continue
		}
		m.Age++
	}

	g.Mines = compact(g.Mines, func(m *Mine) bool { return m.Alive })
}

func checkMineCollisions(g *Game) {
	for i := range g.Mines {
		m := &g.Mines[i]
		if !m.Alive {
			continue
		}
		for j := range g.Enemies {
			e := &g.Enemies[j]
			if !e.Alive {
				continue
			}
			dx := m.X - e.X
			dy := m.Y - e.Y
			if dx*dx+dy*dy < mineEnemyRadSq {
				m.Alive = false
				mineExplode(g, m.X, m.Y)
				break
			}
		}
	}
}

func mineExplode(g *Game, x, y float64) {
	aoeExplode(g, x, y, mineBlastRadSq, EventMineExploded)
}

func deployMine(g *Game) {
	g.Mines = append(g.Mines, Mine{
		X: g.Player.X, Y: g.Player.Y,
		Alive: true,
	})
	g.Events = append(g.Events, Event{
		Type: EventMinePlaced, X: g.Player.X, Y: g.Player.Y,
	})
}

func spawnMineBlast(g *Game, x, y float64) {
	emitBurst(g, x, y, 35, 2.5, 9.0, 30, 50, 3, 8, colorBlastInner)
	emitBurst(g, x, y, 30, 4.0, 9.0, 35, 60, 2, 6, colorMineBlast)
	emitBurst(g, x, y, 20, 0.5, 3.0, 45, 75, 4, 9, ColorSmoke)
}

func drawMines(screen *ebiten.Image, g *Game, ox, oy float64) {
	for i := range g.Mines {
		m := &g.Mines[i]
		cx := float32(m.X + ox)
		cy := float32(m.Y + oy)

		// Slow pulse glow.
		pulse := float32(0.5 + 0.3*math.Sin(float64(m.Age)*0.08))
		glowR := float32(MineRadius+6) * pulse
		glowCol := colorMinePulse
		glowCol.A = uint8(float32(0x66) * pulse)
		vector.DrawFilledCircle(screen, cx, cy, glowR, glowCol, AntiAlias)

		// Body — filled circle.
		vector.DrawFilledCircle(screen, cx, cy, float32(MineRadius), ColorMine, AntiAlias)

		// Spikes — 8 short lines radiating outward.
		spikeLen := float32(MineRadius * 0.7)
		for s := 0; s < 8; s++ {
			dx, dy := spikeDirs[s][0], spikeDirs[s][1]
			sx := cx + dx*float32(MineRadius)
			sy := cy + dy*float32(MineRadius)
			ex := cx + dx*(float32(MineRadius)+spikeLen)
			ey := cy + dy*(float32(MineRadius)+spikeLen)
			vector.StrokeLine(screen, sx, sy, ex, ey, 2, colorMineSpike, AntiAlias)
		}

		// Center dot — blinks.
		if (m.Age/15)%2 == 0 {
			vector.DrawFilledCircle(screen, cx, cy, 3, ColorHeatHot, AntiAlias)
		}
	}
}

func spawnMinePlacedEffect(g *Game, x, y float64) {
	emitBurst(g, x, y, 8, 1.0, 3.0, 15, 25, 2, 4, ColorMine)
}
