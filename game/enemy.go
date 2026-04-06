package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	EnemyRadius        = 22.0
	EnemyBaseSpeed     = 1.5
	EnemyTurnRateMin   = 0.03 // radians per frame (slow turner)
	EnemyTurnRateMax   = 0.10 // radians per frame (sharp turner)
)

type Enemy struct {
	X, Y     float64
	VX, VY   float64
	Speed    float64
	TurnRate float64 // max radians per frame toward player
	HP       int
	MaxHP    int
	Alive    bool
	FlashFrames int
}

func updateEnemies(g *Game) {
	for i := range g.Enemies {
		e := &g.Enemies[i]
		if !e.Alive {
			continue
		}
		// Steer toward player, limited by turn rate.
		dx := g.Player.X - e.X
		dy := g.Player.Y - e.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 1 {
			desiredAngle := math.Atan2(dy, dx)
			currentAngle := math.Atan2(e.VY, e.VX)
			diff := desiredAngle - currentAngle
			// Normalize to [-pi, pi].
			for diff > math.Pi {
				diff -= 2 * math.Pi
			}
			for diff < -math.Pi {
				diff += 2 * math.Pi
			}
			// Clamp turn.
			if diff > e.TurnRate {
				diff = e.TurnRate
			} else if diff < -e.TurnRate {
				diff = -e.TurnRate
			}
			newAngle := currentAngle + diff
			e.VX = math.Cos(newAngle) * e.Speed
			e.VY = math.Sin(newAngle) * e.Speed
		}
		e.X += e.VX
		e.Y += e.VY

		if e.FlashFrames > 0 {
			e.FlashFrames--
		}
	}
	// Compact.
	n := 0
	for i := range g.Enemies {
		if g.Enemies[i].Alive {
			g.Enemies[n] = g.Enemies[i]
			n++
		}
	}
	g.Enemies = g.Enemies[:n]
}

func spawnEnemyThrustParticles(g *Game) {
	for i := range g.Enemies {
		e := &g.Enemies[i]
		if !e.Alive {
			continue
		}
		speed := math.Sqrt(e.VX*e.VX + e.VY*e.VY)
		if speed < 0.5 {
			continue
		}
		// Exhaust opposite to movement direction.
		dirX := -e.VX / speed
		dirY := -e.VY / speed
		spawnThrustParticles(g, e.X+dirX*EnemyRadius, e.Y+dirY*EnemyRadius, dirX, dirY, ColorEnemy)
	}
}

func drawEnemies(screen *ebiten.Image, g *Game, ox, oy float64) {
	// During respawn freeze, enemies blink — faster as timer approaches 0.
	respawnBlink := false
	if g.State == StateRespawn {
		// Blink period shrinks from 30 frames down to 6 as timer runs out.
		frac := float64(g.RespawnTimer) / float64(RespawnFreeze)
		period := int(6 + 24*frac) // 30 at start → 6 near end
		if period < 2 {
			period = 2
		}
		respawnBlink = (g.RespawnTimer/period)%2 == 0
	}

	for i := range g.Enemies {
		e := &g.Enemies[i]

		if respawnBlink {
			continue // hidden during this blink phase
		}

		cx := float32(e.X + ox)
		cy := float32(e.Y + oy)

		col := ColorEnemy
		if e.MaxHP > 1 {
			hpFrac := float32(e.HP) / float32(e.MaxHP)
			col = lerpColor(ColorEnemyHurt, ColorEnemy, hpFrac)
		}
		if e.FlashFrames > 0 {
			col = ColorUI // white flash
		}

		// Outer glow circle.
		r := float32(EnemyRadius)
		vector.StrokeCircle(screen, cx, cy, r+3, 4, col, false)

		angle := math.Atan2(e.VY, e.VX)
		drawPolygon(screen, cx, cy, r, 3, angle, 4, col)
	}
}
