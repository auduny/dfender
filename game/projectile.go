package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ProjectileRadius = 5.0
)

type Projectile struct {
	X, Y   float32
	VX, VY float32
	Alive  bool
}

func updateProjectiles(g *Game) {
	for i := range g.Projectiles {
		p := &g.Projectiles[i]
		if !p.Alive {
			continue
		}
		p.X += p.VX
		p.Y += p.VY

		// Detect wall hit at arena boundary.
		if p.X < ArenaLeft() || p.X > ArenaRight() ||
			p.Y < ArenaTop() || p.Y > ArenaBottom() {
			// Clamp impact position to arena edge.
			ix := max(ArenaLeft(), min(p.X, ArenaRight()))
			iy := max(ArenaTop(), min(p.Y, ArenaBottom()))
			g.Events = append(g.Events, Event{
				Type: EventProjectileWallHit,
				X:    ix,
				Y:    iy,
			})
			p.Alive = false
		}
	}
	g.Projectiles = compact(g.Projectiles, func(p *Projectile) bool { return p.Alive })
}

func drawProjectiles(screen *ebiten.Image, g *Game, ox, oy float32) {
	for i := range g.Projectiles {
		p := &g.Projectiles[i]
		cx := p.X + ox
		cy := p.Y + oy
		// Outer glow.
		vector.DrawFilledCircle(screen, cx, cy, ProjectileRadius+3, ColorHeatCool, AntiAlias)
		// Bright core.
		vector.DrawFilledCircle(screen, cx, cy, ProjectileRadius, ColorProjectile, AntiAlias)
		// Trail — thicker line behind.
		tx := p.X - p.VX*0.8 + ox
		ty := p.Y - p.VY*0.8 + oy
		vector.StrokeLine(screen, cx, cy, tx, ty, 4, ColorHeatCool, AntiAlias)
	}
}
