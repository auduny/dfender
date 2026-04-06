package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	PlayerRadius     = 32.0
	ThrustForce      = 0.15
	WallBounceDamp   = 0.5
	WallDeathSpeed   = 8.0 // speed above which wall collision kills
)

const (
	InvulnDuration = 240 // 4 seconds at 60 TPS
)

type Player struct {
	X, Y         float64
	VX, VY       float64
	Alive        bool
	InvulnFrames int // frames of invulnerability remaining
}

func NewPlayer(x, y float64) Player {
	return Player{X: x, Y: y, Alive: true}
}

func (p *Player) Speed() float64 {
	return math.Sqrt(p.VX*p.VX + p.VY*p.VY)
}

func (p *Player) Update() {
	if !p.Alive {
		return
	}

	// Thruster input.
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		p.VY -= ThrustForce
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		p.VY += ThrustForce
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		p.VX -= ThrustForce
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		p.VX += ThrustForce
	}

	// Integrate position.
	p.X += p.VX
	p.Y += p.VY

	// Tick down invulnerability.
	if p.InvulnFrames > 0 {
		p.InvulnFrames--
	}
}

// CheckWalls handles wall collision, emitting events as needed.
func (p *Player) CheckWalls(g *Game) {
	if !p.Alive {
		return
	}

	left := ArenaLeft() + PlayerRadius
	right := ArenaRight() - PlayerRadius
	top := ArenaTop() + PlayerRadius
	bottom := ArenaBottom() - PlayerRadius

	collided := false
	impactSpeed := 0.0

	if p.X < left {
		impactSpeed = math.Max(impactSpeed, math.Abs(p.VX))
		p.X = left
		p.VX = -p.VX * WallBounceDamp
		collided = true
	} else if p.X > right {
		impactSpeed = math.Max(impactSpeed, math.Abs(p.VX))
		p.X = right
		p.VX = -p.VX * WallBounceDamp
		collided = true
	}

	if p.Y < top {
		impactSpeed = math.Max(impactSpeed, math.Abs(p.VY))
		p.Y = top
		p.VY = -p.VY * WallBounceDamp
		collided = true
	} else if p.Y > bottom {
		impactSpeed = math.Max(impactSpeed, math.Abs(p.VY))
		p.Y = bottom
		p.VY = -p.VY * WallBounceDamp
		collided = true
	}

	if collided {
		if impactSpeed >= WallDeathSpeed && p.InvulnFrames <= 0 {
			g.Events = append(g.Events, Event{Type: EventWallDeath, X: p.X, Y: p.Y})
			p.Alive = false
		} else {
			g.Events = append(g.Events, Event{Type: EventWallBounce, X: p.X, Y: p.Y, Value: impactSpeed})
		}
	}
}

// SpawnThrustParticles emits exhaust when thrusters are active.
func (p *Player) SpawnThrustParticles(g *Game) {
	if !p.Alive {
		return
	}
	exhaust := ColorPlayer
	if ebiten.IsKeyPressed(ebiten.KeyW) {
		spawnThrustParticles(g, p.X, p.Y+PlayerRadius, 0, 1, exhaust)
	}
	if ebiten.IsKeyPressed(ebiten.KeyS) {
		spawnThrustParticles(g, p.X, p.Y-PlayerRadius, 0, -1, exhaust)
	}
	if ebiten.IsKeyPressed(ebiten.KeyA) {
		spawnThrustParticles(g, p.X+PlayerRadius, p.Y, 1, 0, exhaust)
	}
	if ebiten.IsKeyPressed(ebiten.KeyD) {
		spawnThrustParticles(g, p.X-PlayerRadius, p.Y, -1, 0, exhaust)
	}
}

func (p *Player) Draw(screen *ebiten.Image, ox, oy float64, heat float64) {
	if !p.Alive {
		return
	}

	// Flicker when invulnerable (skip drawing every other 4-frame block).
	if p.InvulnFrames > 0 && (p.InvulnFrames/4)%2 == 0 {
		return
	}

	cx := float32(p.X + ox)
	cy := float32(p.Y + oy)
	r := float32(PlayerRadius)

	// Tint ship from gold toward red as heat rises.
	shipColor := lerpColor(ColorPlayer, ColorHeatHot, float32(heat))

	// Outer glow ring (dimmer, larger).
	vector.StrokeCircle(screen, cx, cy, r+4, 6, ColorBorderDim, false)

	drawPolygon(screen, cx, cy, r, 6, -math.Pi/2, 4, shipColor)

	// Inner dot.
	vector.DrawFilledCircle(screen, cx, cy, 5, shipColor, false)
}
