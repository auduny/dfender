package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type PowerUpType int

const (
	PowerUpShield    PowerUpType = iota
	PowerUpGuns
	PowerUpMissile
	PowerUpSupercool
	PowerUpCount // must be last — used for random selection
)

const (
	PowerUpRadius     = 18.0
	PowerUpLifetime   = 600 // 10s at 60fps
	PowerUpBlinkStart = 180 // last 3s
	PowerUpDropChance = 0.20
	PowerUpRotSpeed   = 0.03 // radians/frame
	PowerUpBobSpeed   = 0.05 // bob animation speed
	PowerUpBobAmount  = 3.0  // bob amplitude in pixels
	GunsBuffDuration      = 1200 // 20s at 60fps
	SupercoolBuffDuration = 900  // 15s at 60fps
	SupercoolHeatCap      = 0.95 // heat cannot exceed this while active
)

// PlayerPowerUps tracks the player's active power-up state.
type PlayerPowerUps struct {
	Shield        bool
	GunsTimer     int // frames remaining (0 = inactive)
	MissileCount  int
	SupercoolTimer int // frames remaining (0 = inactive)
}

type PowerUp struct {
	X, Y     float64
	Type     PowerUpType
	Life     int // frames remaining (0 = dead)
	Rotation float64
}

func updatePowerUps(g *Game) {
	for i := range g.PowerUps {
		pu := &g.PowerUps[i]
		if pu.Life <= 0 {
			continue
		}
		pu.Life--
		pu.Rotation += PowerUpRotSpeed
	}
	// Compact.
	n := 0
	for i := range g.PowerUps {
		if g.PowerUps[i].Life > 0 {
			g.PowerUps[n] = g.PowerUps[i]
			n++
		}
	}
	g.PowerUps = g.PowerUps[:n]
}

func drawPowerUps(screen *ebiten.Image, g *Game, ox, oy float64) {
	for i := range g.PowerUps {
		pu := &g.PowerUps[i]

		// Blink when about to despawn.
		if pu.Life < PowerUpBlinkStart {
			period := 8
			if pu.Life < 60 {
				period = 4
			}
			if (pu.Life/period)%2 == 0 {
				continue
			}
		}

		// Bob animation.
		bob := math.Sin(float64(g.Tick)*PowerUpBobSpeed) * PowerUpBobAmount

		cx := float32(pu.X + ox)
		cy := float32(pu.Y + oy + bob)
		r := float32(PowerUpRadius)

		var col color.RGBA
		var sides int

		switch pu.Type {
		case PowerUpShield:
			col = ColorShield  // green
			sides = 6          // hexagon
		case PowerUpGuns:
			col = ColorPlayer  // amber
			sides = 5          // pentagon
		case PowerUpMissile:
			col = ColorHeatHot // red
			sides = 4          // diamond
		case PowerUpSupercool:
			col = ColorSupercool // blue
			sides = 7            // heptagon
		}

		// Outer glow.
		vector.StrokeCircle(screen, cx, cy, r+4, 2, col, AntiAlias)

		// Shape.
		drawPolygon(screen, cx, cy, r, sides, pu.Rotation, 3, col)

		// Inner dot.
		vector.DrawFilledCircle(screen, cx, cy, 4, col, AntiAlias)
	}
}

// drawShieldOverlay draws a hexagonal ring around the player when shield is held.
func drawShieldOverlay(screen *ebiten.Image, g *Game, ox, oy float64) {
	if !g.PlayerPowerUps.Shield || !g.Player.Alive {
		return
	}
	// Flicker with player during invuln.
	if g.Player.InvulnFrames > 0 && (g.Player.InvulnFrames/4)%2 == 0 {
		return
	}
	cx := float32(g.Player.X + ox)
	cy := float32(g.Player.Y + oy)
	r := float32(PlayerRadius + 10)
	// Slow pulse.
	pulse := float32(0.5 + 0.5*math.Sin(float64(g.Tick)*0.08))
	dimShield := color.RGBA{0x1A, 0x6E, 0x2A, 0xFF}
	col := lerpColor(dimShield, ColorShield, pulse)
	drawPolygon(screen, cx, cy, r, 6, float64(g.Tick)*0.02, 2, col)
}

func spawnPowerUpDrop(g *Game, x, y float64, waveNumber int) {
	if waveNumber < 2 {
		return
	}
	if rand.Float64() > PowerUpDropChance {
		return
	}

	puType := PowerUpType(rand.Intn(int(PowerUpCount)))

	g.PowerUps = append(g.PowerUps, PowerUp{
		X: x, Y: y,
		Type: puType,
		Life: PowerUpLifetime,
	})
}
