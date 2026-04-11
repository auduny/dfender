package game

import (
	"image/color"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// drawEnemyShape draws a single enemy: shield ring, inner body, outer triangle.
func drawEnemyShape(screen *ebiten.Image, cx, cy, r float32, angle float32, shieldCol, innerCol color.RGBA) {
	vector.StrokeCircle(screen, cx, cy, r+3, 4, shieldCol, AntiAlias)
	vector.DrawFilledCircle(screen, cx, cy, r*0.55, innerCol, AntiAlias)
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
	EnemyGreenEvadeAngle   = 15.0 * pi32 / 180.0 // 15 degrees offset
	EnemyGreenSwitchRate   = 60                    // frames between direction switches
	EnemyGreenTeleportMin  = 45                    // min frames between teleports (~0.75s)
	EnemyGreenTeleportMax  = 90                    // max frames between teleports (1.5s)
	EnemyGreenTeleportDist = 120.0                 // pixels to teleport sideways
)

type Enemy struct {
	X, Y        float32
	VX, VY      float32
	Speed       float32
	TurnRate    float32 // max radians per frame toward player
	HP          int
	MaxHP       int
	Alive       bool
	FlashFrames int
	Type        EnemyType
	Accel       float32 // per-frame speed increase (Red type)
	MaxSpeed    float32 // speed cap (Red type)
	EvadeTick     int     // frame counter for Green evasion switching
	EvadeSign     float32 // +1 or -1 for Green evasion direction
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
		dist := sqrt32(dx*dx + dy*dy)
		if dist > 1 {
			desiredAngle := atan232(dy, dx)

			// Green: offset the desired angle to weave.
			if e.Type == EnemyGreen {
				desiredAngle += EnemyGreenEvadeAngle * e.EvadeSign
			}

			currentAngle := atan232(e.VY, e.VX)
			diff := desiredAngle - currentAngle
			// Normalize to [-pi, pi].
			for diff > pi32 {
				diff -= 2 * pi32
			}
			for diff < -pi32 {
				diff += 2 * pi32
			}
			// Clamp turn.
			if diff > e.TurnRate {
				diff = e.TurnRate
			} else if diff < -e.TurnRate {
				diff = -e.TurnRate
			}
			newAngle := currentAngle + diff
			e.VX = cos32(newAngle) * e.Speed
			e.VY = sin32(newAngle) * e.Speed
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
	g.Enemies = compact(g.Enemies, func(e *Enemy) bool { return e.Alive })
}

// teleportBrain blinks a Brain enemy sideways (perpendicular to heading).
// Picks a random direction, clamps to arena bounds.
func teleportBrain(g *Game, e *Enemy) {
	// Perpendicular to current heading.
	heading := atan232(e.VY, e.VX)
	perpAngle := heading + pi32/2
	if rand.Intn(2) == 0 {
		perpAngle = heading - pi32/2
	}

	newX := e.X + cos32(perpAngle)*EnemyGreenTeleportDist
	newY := e.Y + sin32(perpAngle)*EnemyGreenTeleportDist

	// Clamp to arena bounds with margin so they don't land in a wall.
	margin := float32(EnemyRadius + 5)
	newX = max(ArenaLeft()+margin, min(newX, ArenaRight()-margin))
	newY = max(ArenaTop()+margin, min(newY, ArenaBottom()-margin))

	// Particles at old position (vanish).
	spawnExplosion(g, e.X, e.Y, ColorEnemyGreen, 12)

	e.X = newX
	e.Y = newY

	// Particles at new position (appear).
	spawnExplosion(g, e.X, e.Y, ColorEnemyGreen, 12)
}

// enemyHitsWall returns true if the enemy is outside the arena bounds,
// excluding the gate openings where enemies enter.
func enemyHitsWall(e *Enemy) bool {
	halfGate := float32(GateWidth) / 2
	r := float32(EnemyRadius)
	top, bottom := ArenaTop(), ArenaBottom()
	left, right := ArenaLeft(), ArenaRight()
	midX := float32(ScreenWidth) / 2
	midY := float32(ScreenHeight) / 2

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
		speed := sqrt32(e.VX*e.VX + e.VY*e.VY)
		if speed < 0.5 {
			continue
		}
		// Exhaust opposite to movement direction.
		dirX := -e.VX / speed
		dirY := -e.VY / speed
		spawnThrustParticles(g, e.X+dirX*EnemyRadius, e.Y+dirY*EnemyRadius, dirX, dirY, ColorEnemy)
	}
}

func drawEnemies(screen *ebiten.Image, g *Game, ox, oy float32) {
	// During respawn freeze, enemies blink — faster as timer approaches 0.
	respawnBlink := false
	if g.State == StateRespawn {
		// Blink period shrinks from 30 frames down to 6 as timer runs out.
		frac := float32(g.RespawnTimer) / float32(RespawnFreeze)
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

		cx := e.X + ox
		cy := e.Y + oy

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
		angle := atan232(e.VY, e.VX)
		drawEnemyShape(screen, cx, cy, r, angle, shieldCol, innerCol)
	}
}
