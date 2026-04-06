package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawEnemyShape draws a single enemy: shield ring, inner body, outer triangle.
func drawEnemyShape(screen *ebiten.Image, cx, cy, r float32, angle float64, shieldCol, innerCol color.RGBA) {
	vector.StrokeCircle(screen, cx, cy, r+3, 4, shieldCol, false)
	vector.DrawFilledCircle(screen, cx, cy, r*0.55, innerCol, false)
	drawPolygon(screen, cx, cy, r, 3, angle, 2, shieldCol)
}

// EnemyType determines behavior and inner color.
type EnemyType int

const (
	EnemyNormal EnemyType = iota // white inner, balanced stats
	EnemyRed                     // red inner, wide turns, accelerates
	EnemyGreen                   // green inner, evasive steering
)

const (
	EnemyRadius = 22.0

	// Normal: moderate speed, fixed turn rate.
	EnemyNormalSpeed    = 1.5
	EnemyNormalTurnRate = 0.04

	// Red: starts slow, accelerates, wide turning radius.
	EnemyRedBaseSpeed    = 1.0
	EnemyRedAccel        = 0.02 // speed gained per frame
	EnemyRedMaxSpeed     = 4.5
	EnemyRedTurnRate     = 0.015

	// Green (Brain): moderate speed, sharp turns, evasive offset, teleport.
	EnemyGreenSpeed        = 1.8
	EnemyGreenTurnRate     = 0.06
	EnemyGreenEvadeAngle   = 15.0 * math.Pi / 180.0 // 15 degrees offset
	EnemyGreenSwitchRate   = 60                       // frames between direction switches
	EnemyGreenTeleportMin  = 120                      // min frames between teleports (2s)
	EnemyGreenTeleportMax  = 180                      // max frames between teleports (3s)
	EnemyGreenTeleportDist = 120.0                    // pixels to teleport sideways
)

type Enemy struct {
	X, Y        float64
	VX, VY      float64
	Speed       float64
	TurnRate    float64 // max radians per frame toward player
	HP          int
	MaxHP       int
	Alive       bool
	FlashFrames int
	Type        EnemyType
	Accel       float64 // per-frame speed increase (Red type)
	MaxSpeed    float64 // speed cap (Red type)
	EvadeTick     int     // frame counter for Green evasion switching
	EvadeSign     float64 // +1 or -1 for Green evasion direction
	TeleportTimer int     // frames until next teleport (Green/Brain)
}

func updateEnemies(g *Game) {
	for i := range g.Enemies {
		e := &g.Enemies[i]
		if !e.Alive {
			continue
		}

		// Red: accelerate each frame up to max speed.
		if e.Type == EnemyRed && e.Speed < e.MaxSpeed {
			e.Speed += e.Accel
			if e.Speed > e.MaxSpeed {
				e.Speed = e.MaxSpeed
			}
		}

		// Green (Brain): flip evasion direction periodically + teleport.
		if e.Type == EnemyGreen {
			e.EvadeTick++
			if e.EvadeTick >= EnemyGreenSwitchRate {
				e.EvadeTick = 0
				e.EvadeSign = -e.EvadeSign
			}
			e.TeleportTimer--
			if e.TeleportTimer <= 0 {
				e.TeleportTimer = EnemyGreenTeleportMin + rand.Intn(EnemyGreenTeleportMax-EnemyGreenTeleportMin)
				teleportBrain(g, e)
			}
		}

		// Steer toward player (with evasion offset for Green), limited by turn rate.
		dx := g.Player.X - e.X
		dy := g.Player.Y - e.Y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 1 {
			desiredAngle := math.Atan2(dy, dx)

			// Green: offset the desired angle to weave.
			if e.Type == EnemyGreen {
				desiredAngle += EnemyGreenEvadeAngle * e.EvadeSign
			}

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

		// Wall collision: enemies explode on contact, but not inside gate openings.
		if enemyHitsWall(e) {
			e.Alive = false
			g.Events = append(g.Events, Event{
				Type: EventEnemyWallDeath, X: e.X, Y: e.Y,
			})
		}

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

// teleportBrain blinks a Brain enemy sideways (perpendicular to heading).
// Picks a random direction, clamps to arena bounds.
func teleportBrain(g *Game, e *Enemy) {
	// Perpendicular to current heading.
	heading := math.Atan2(e.VY, e.VX)
	perpAngle := heading + math.Pi/2
	if rand.Intn(2) == 0 {
		perpAngle = heading - math.Pi/2
	}

	newX := e.X + math.Cos(perpAngle)*EnemyGreenTeleportDist
	newY := e.Y + math.Sin(perpAngle)*EnemyGreenTeleportDist

	// Clamp to arena bounds with margin so they don't land in a wall.
	margin := EnemyRadius + 5
	newX = math.Max(ArenaLeft()+margin, math.Min(newX, ArenaRight()-margin))
	newY = math.Max(ArenaTop()+margin, math.Min(newY, ArenaBottom()-margin))

	// Particles at old position (vanish).
	spawnExplosion(g, e.X, e.Y, ColorEnemyGreen, 8)

	e.X = newX
	e.Y = newY

	// Particles at new position (appear).
	spawnExplosion(g, e.X, e.Y, ColorEnemyGreen, 8)
}

// enemyHitsWall returns true if the enemy is outside the arena bounds,
// excluding the gate openings where enemies enter.
func enemyHitsWall(e *Enemy) bool {
	halfGate := float64(GateWidth) / 2
	r := EnemyRadius
	top, bottom := ArenaTop(), ArenaBottom()
	left, right := ArenaLeft(), ArenaRight()
	midX := float64(ScreenWidth) / 2
	midY := float64(ScreenHeight) / 2

	// Each wall: if enemy is beyond it, check whether it's inside the gate opening.
	if e.Y-r < top && (e.X < midX-halfGate || e.X > midX+halfGate) {
		return true
	}
	if e.Y+r > bottom && (e.X < midX-halfGate || e.X > midX+halfGate) {
		return true
	}
	if e.X-r < left && (e.Y < midY-halfGate || e.Y > midY+halfGate) {
		return true
	}
	if e.X+r > right && (e.Y < midY-halfGate || e.Y > midY+halfGate) {
		return true
	}
	return false
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

		// Shield ring color: cyan→hotpink based on HP.
		shieldCol := ColorEnemy
		if e.MaxHP > 1 {
			hpFrac := float32(e.HP) / float32(e.MaxHP)
			shieldCol = lerpColor(ColorEnemyHurt, ColorEnemy, hpFrac)
		}
		if e.FlashFrames > 0 {
			shieldCol = ColorUI // white flash
		}

		// Inner color determined by enemy type (does not change with damage).
		innerCol := ColorEnemyNormal
		switch e.Type {
		case EnemyRed:
			innerCol = ColorEnemyRed
		case EnemyGreen:
			innerCol = ColorEnemyGreen
		}

		r := float32(EnemyRadius)
		angle := math.Atan2(e.VY, e.VX)
		drawEnemyShape(screen, cx, cy, r, angle, shieldCol, innerCol)
	}
}
