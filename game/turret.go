package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	TurretRotSpeed   = math.Pi // radians per second → 180 deg/s
	TurretLength     = 38.0
	FireRate         = 6  // frames between shots (10/sec at 60fps)
	GunsFireRate     = 3  // frames between shots with guns powerup
	GunsSpread       = 0.08 // radians spread for double barrel
	ProjectileSpeed  = 12.0
	HeatPerShot      = 0.05  // ~2 sec to overheat at 10 rps
	HeatDecay        = 0.004 // ~4 sec to cool from full
	CooldownTime     = 120   // frames (2 sec)
)

type Turret struct {
	Angle     float64 // radians, 0 = up
	Heat      float64 // 0..1
	Cooldown  int     // frames remaining in overheat lockout
	FireTimer int     // frames until next shot allowed
}

func NewTurret() Turret {
	return Turret{Angle: -math.Pi / 2} // pointing up
}

func (t *Turret) Update(g *Game) {
	if !g.Player.Alive {
		return
	}

	// Rotation.
	rotSpeed := TurretRotSpeed / 60.0 // per frame
	if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) {
		t.Angle -= rotSpeed
	}
	if ebiten.IsKeyPressed(ebiten.KeyArrowRight) {
		t.Angle += rotSpeed
	}

	// Cooldown.
	if t.Cooldown > 0 {
		t.Cooldown--
		t.Heat = float64(t.Cooldown) / float64(CooldownTime)
		if t.Cooldown == 0 {
			t.Heat = 0
		} else if t.Cooldown%20 == 0 {
			g.Events = append(g.Events, Event{Type: EventOverheatWarning})
		}
		return
	}

	// Heat decay.
	if t.Heat > 0 {
		t.Heat -= HeatDecay
		if t.Heat < 0 {
			t.Heat = 0
		}
	}

	// Fire timer.
	if t.FireTimer > 0 {
		t.FireTimer--
	}

	// Missile firing (E key).
	if inpututil.IsKeyJustPressed(ebiten.KeyE) && g.PlayerPowerUps.MissileCount > 0 {
		g.PlayerPowerUps.MissileCount--
		fireMissile(g)
	}

	// Determine fire rate based on guns buff.
	rate := FireRate
	gunsActive := g.PlayerPowerUps.GunsTimer > 0
	if gunsActive {
		rate = GunsFireRate
	}

	// Firing.
	if ebiten.IsKeyPressed(ebiten.KeySpace) && t.FireTimer == 0 && t.Cooldown == 0 {
		t.FireTimer = rate
		t.Heat += HeatPerShot
		if g.PlayerPowerUps.SupercoolTimer > 0 {
			if t.Heat > SupercoolHeatCap {
				t.Heat = SupercoolHeatCap
			}
		} else if t.Heat >= 1.0 {
			t.Heat = 1.0
			t.Cooldown = CooldownTime
			g.Events = append(g.Events, Event{Type: EventOverheat, X: g.Player.X, Y: g.Player.Y})
			return
		}
		// Spawn projectile(s).
		dx := math.Cos(t.Angle)
		dy := math.Sin(t.Angle)
		spawnX := g.Player.X + dx*TurretLength
		spawnY := g.Player.Y + dy*TurretLength

		if gunsActive {
			// Double barrel: two projectiles with spread.
			for _, offset := range []float64{-GunsSpread, GunsSpread} {
				a := t.Angle + offset
				pdx := math.Cos(a)
				pdy := math.Sin(a)
				g.Projectiles = append(g.Projectiles, Projectile{
					X: spawnX, Y: spawnY,
					VX: pdx * ProjectileSpeed, VY: pdy * ProjectileSpeed,
					Alive: true,
				})
			}
		} else {
			g.Projectiles = append(g.Projectiles, Projectile{
				X: spawnX, Y: spawnY,
				VX: dx * ProjectileSpeed, VY: dy * ProjectileSpeed,
				Alive: true,
			})
		}
		g.Events = append(g.Events, Event{Type: EventFired, X: spawnX, Y: spawnY, Value: t.Heat})
	}
}

func (t *Turret) Draw(screen *ebiten.Image, g *Game, ox, oy float64) {
	if !g.Player.Alive {
		return
	}
	cx := float32(g.Player.X + ox)
	cy := float32(g.Player.Y + oy)
	dx := float32(math.Cos(t.Angle))
	dy := float32(math.Sin(t.Angle))

	// Barrel.
	endX := cx + dx*TurretLength
	endY := cy + dy*TurretLength

	// Color shifts with heat.
	barrelColor := lerpColor(ColorPlayer, ColorHeatHot, float32(t.Heat))
	vector.StrokeLine(screen, cx, cy, endX, endY, 5, barrelColor, AntiAlias)

	// Muzzle dot.
	vector.DrawFilledCircle(screen, endX, endY, 5, barrelColor, AntiAlias)

	// Aim line (subtle).
	aimLen := float32(60.0)
	aimEndX := endX + dx*aimLen
	aimEndY := endY + dy*aimLen
	aimColor := ColorBorderDim
	vector.StrokeLine(screen, endX, endY, aimEndX, aimEndY, 1, aimColor, AntiAlias)
}
