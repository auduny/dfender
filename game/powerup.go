package game

import (
	"image/color"
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
	PowerUpMine
	PowerUpExtraLife
	PowerUpCount // must be last — used for random selection
)

const (
	PowerUpRadius     = 18.0
	PowerUpLifetime   = 600 // 10s at 60fps
	PowerUpBlinkStart = 180 // last 3s
	PowerUpDropChance = 0.30
	PowerUpRotSpeed   = 0.03 // radians/frame
	PowerUpBobSpeed   = 0.05 // bob animation speed
	PowerUpBobAmount  = 3.0  // bob amplitude in pixels
	GunsBuffDuration      = 1200 // 20s at 60fps
	SupercoolBuffDuration = 3600 // 60s at 60fps
	SupercoolHeatCap      = 0.95 // heat cannot exceed this while active

	ExtraLifeDropChance = 0.05 // 5% per kill — much rarer than normal drops
	CornerPowerUpWave   = 10   // wave at which corner powerups start spawning
	CornerInset         = 80.0 // pixels from arena edge for corner placement
)

// PlayerPowerUps tracks the player's active power-up state.
type PlayerPowerUps struct {
	Shield        bool
	GunsTimer     int // frames remaining (0 = inactive)
	MissileCount  int
	MineCount     int
	SupercoolTimer int // frames remaining (0 = inactive)
}

type PowerUp struct {
	X, Y       float32
	Type       PowerUpType
	Life       int // frames remaining (0 = dead)
	Rotation   float32
	Persistent bool // if true, never expires on its own
}

func updatePowerUps(g *Game) {
	for i := range g.PowerUps {
		pu := &g.PowerUps[i]
		if pu.Life <= 0 {
			continue
		}
		if !pu.Persistent {
			pu.Life--
		}
		pu.Rotation += PowerUpRotSpeed
	}
	g.PowerUps = compact(g.PowerUps, func(p *PowerUp) bool { return p.Life > 0 })
}

func drawPowerUps(screen *ebiten.Image, g *Game, ox, oy float32) {
	for i := range g.PowerUps {
		pu := &g.PowerUps[i]

		// Blink when about to despawn (persistent powerups never blink).
		if !pu.Persistent && pu.Life < PowerUpBlinkStart {
			period := 8
			if pu.Life < 60 {
				period = 4
			}
			if (pu.Life/period)%2 == 0 {
				continue
			}
		}

		// Bob animation.
		bob := sin32(float32(g.Tick)*PowerUpBobSpeed) * PowerUpBobAmount

		cx := pu.X + ox
		cy := pu.Y + oy + bob
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
		case PowerUpMine:
			col = ColorMine // orange
			sides = 8       // octagon
		case PowerUpExtraLife:
			col = ColorExtraLife // magenta-pink
			sides = 0           // special: heart shape
		}

		// Outer glow.
		vector.StrokeCircle(screen, cx, cy, r+4, 2, col, AntiAlias)

		// Shape.
		if sides > 0 {
			drawPolygon(screen, cx, cy, r, sides, pu.Rotation, 3, col)
		} else {
			drawHeart(screen, cx, cy, r, pu.Rotation, 3, col)
		}

		// Inner dot.
		vector.DrawFilledCircle(screen, cx, cy, 4, col, AntiAlias)
	}
}

// drawShieldOverlay draws a hexagonal ring around the player when shield is held.
func drawShieldOverlay(screen *ebiten.Image, g *Game, ox, oy float32) {
	if !g.PlayerPowerUps.Shield || !g.Player.Alive {
		return
	}
	// Flicker with player during invuln.
	if g.Player.InvulnFrames > 0 && (g.Player.InvulnFrames/4)%2 == 0 {
		return
	}
	cx := g.Player.X + ox
	cy := g.Player.Y + oy
	r := float32(PlayerRadius + 10)
	// Slow pulse.
	pulse := 0.5 + 0.5*sin32(float32(g.Tick)*0.08)
	dimShield := color.RGBA{0x1A, 0x6E, 0x2A, 0xFF}
	col := lerpColor(dimShield, ColorShield, pulse)
	drawPolygon(screen, cx, cy, r, 6, float32(g.Tick)*0.02, 2, col)
}

// powerUpUnlockWave maps each power-up type to the wave it first becomes available.
var powerUpUnlockWave = [PowerUpCount]int{
	PowerUpShield:    1,
	PowerUpGuns:      2,
	PowerUpSupercool: 3,
	PowerUpMissile:   4,
	PowerUpMine:      5,
	PowerUpExtraLife: 10,
}

func spawnPowerUpDrop(g *Game, x, y float32, waveNumber int) {
	// Extra life: separate rare roll, max one per wave.
	if waveNumber >= powerUpUnlockWave[PowerUpExtraLife] &&
		!g.Wave.ExtraLifeDropped &&
		rand.Float64() < ExtraLifeDropChance {
		g.Wave.ExtraLifeDropped = true
		g.PowerUps = append(g.PowerUps, PowerUp{
			X: x, Y: y,
			Type: PowerUpExtraLife,
			Life: PowerUpLifetime,
		})
		return
	}

	if rand.Float64() > PowerUpDropChance {
		return
	}

	// Build pool of unlocked power-up types for this wave (excluding extra life).
	var pool [PowerUpCount]PowerUpType
	n := 0
	for t := PowerUpType(0); t < PowerUpCount; t++ {
		if t == PowerUpExtraLife {
			continue
		}
		if waveNumber >= powerUpUnlockWave[t] {
			pool[n] = t
			n++
		}
	}
	if n == 0 {
		return
	}

	puType := pool[rand.Intn(n)]

	g.PowerUps = append(g.PowerUps, PowerUp{
		X: x, Y: y,
		Type: puType,
		Life: PowerUpLifetime,
	})
}

// spawnCornerPowerUps places guns and supercool in diagonally opposite corners.
// Called at the start of each wave from CornerPowerUpWave onward.
func spawnCornerPowerUps(g *Game) {
	if g.Wave.Number < CornerPowerUpWave {
		return
	}

	left := float32(ArenaMargin + CornerInset)
	right := float32(ScreenWidth - ArenaMargin - CornerInset)
	top := float32(ArenaMargin + StatusBarHeight + CornerInset)
	bottom := float32(ScreenHeight - ArenaMargin - CornerInset)

	// Alternate diagonal each wave so the player can't camp one route.
	topLeft, bottomRight := PowerUpGuns, PowerUpSupercool
	if g.Wave.Number%2 != 0 {
		topLeft, bottomRight = bottomRight, topLeft
	}
	g.PowerUps = append(g.PowerUps,
		PowerUp{X: left, Y: top, Type: PowerUpType(topLeft), Life: 1, Persistent: true},
		PowerUp{X: right, Y: bottom, Type: PowerUpType(bottomRight), Life: 1, Persistent: true},
	)
}

// drawHeart draws a heart outline at (cx, cy) with the given size, rotation and style.
func drawHeart(screen *ebiten.Image, cx, cy, size, rotation, thickness float32, col color.RGBA) {
	const steps = 24
	s := size / 17 // heart formula spans roughly -17..17
	cosR, sinR := cos32(rotation), sin32(rotation)
	var pts [steps]struct{ x, y float32 }
	for i := range steps {
		t := float32(i) * 2 * pi32 / steps
		// Parametric heart: x = 16 sin³(t), y = 13cos(t) - 5cos(2t) - 2cos(3t) - cos(4t)
		hx := 16 * sin32(t) * sin32(t) * sin32(t)
		hy := -(13*cos32(t) - 5*cos32(2*t) - 2*cos32(3*t) - cos32(4*t))
		rx := hx*cosR - hy*sinR
		ry := hx*sinR + hy*cosR
		pts[i].x = cx + rx*s
		pts[i].y = cy + ry*s
	}
	for i := range steps {
		j := (i + 1) % steps
		vector.StrokeLine(screen, pts[i].x, pts[i].y, pts[j].x, pts[j].y, thickness, col, AntiAlias)
	}
}
