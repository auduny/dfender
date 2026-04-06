package game

import (
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	MissileInitialSpeed = 2.0
	MissileMaxSpeed     = 10.0
	MissileAccel        = 0.15
	MissileRadius       = 8.0
	MissileTurnRate     = 0.08 // radians/frame
	MissileMaxCount     = 9
	MissileBlastRadius  = 80.0

	missileBlastRadSq = MissileBlastRadius * MissileBlastRadius
)

var (
	colorMissileNose  = color.RGBA{0xFF, 0xCC, 0x33, 0xFF} // bright gold nose
	colorMissileBody  = color.RGBA{0xFF, 0x55, 0x22, 0xFF} // orange-red body
	colorMissileFlame = color.RGBA{0xFF, 0x88, 0x00, 0xFF} // orange flame
	colorMissileSmoke = color.RGBA{0x99, 0x77, 0x55, 0xCC} // brownish smoke trail
	colorBlastInner   = color.RGBA{0xFF, 0xFF, 0xDD, 0xFF} // white-hot center
	colorBlastOuter   = color.RGBA{0xFF, 0x66, 0x00, 0xFF} // orange ring
)

type Missile struct {
	X, Y  float64
	Angle float64
	Speed float64
	Age   int
	Alive bool
}

func updateMissiles(g *Game) {
	for i := range g.Missiles {
		m := &g.Missiles[i]
		if !m.Alive {
			continue
		}
		m.Age++

		// Accelerate up to max speed.
		if m.Speed < MissileMaxSpeed {
			m.Speed += MissileAccel
			if m.Speed > MissileMaxSpeed {
				m.Speed = MissileMaxSpeed
			}
		}

		// Find nearest enemy in forward hemisphere.
		headX := math.Cos(m.Angle)
		headY := math.Sin(m.Angle)
		bestDist := math.MaxFloat64
		bestAngle := m.Angle
		hasTarget := false

		for j := range g.Enemies {
			e := &g.Enemies[j]
			if !e.Alive {
				continue
			}
			dx := e.X - m.X
			dy := e.Y - m.Y

			// Forward hemisphere check.
			if headX*dx+headY*dy <= 0 {
				continue
			}

			dist := dx*dx + dy*dy
			if dist < bestDist {
				bestDist = dist
				bestAngle = math.Atan2(dy, dx)
				hasTarget = true
			}
		}

		// Steer toward target.
		if hasTarget {
			diff := math.Remainder(bestAngle-m.Angle, 2*math.Pi)
			if diff > MissileTurnRate {
				diff = MissileTurnRate
			} else if diff < -MissileTurnRate {
				diff = -MissileTurnRate
			}
			m.Angle += diff
		}

		vx := math.Cos(m.Angle) * m.Speed
		vy := math.Sin(m.Angle) * m.Speed
		m.X += vx
		m.Y += vy

		// Spawn smoke trail particles every frame.
		spawnMissileTrail(g, m)

		// Wall collision — explode on impact.
		if m.X < ArenaLeft() || m.X > ArenaRight() ||
			m.Y < ArenaTop() || m.Y > ArenaBottom() {
			ix := math.Max(ArenaLeft(), math.Min(m.X, ArenaRight()))
			iy := math.Max(ArenaTop(), math.Min(m.Y, ArenaBottom()))
			m.Alive = false
			missileExplode(g, ix, iy)
		}
	}

	// Compact.
	n := 0
	for i := range g.Missiles {
		if g.Missiles[i].Alive {
			g.Missiles[n] = g.Missiles[i]
			n++
		}
	}
	g.Missiles = g.Missiles[:n]
}

func checkMissileCollisions(g *Game) {
	for i := range g.Missiles {
		m := &g.Missiles[i]
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
			if dx*dx+dy*dy < missileEnemyRadSq {
				m.Alive = false
				missileExplode(g, m.X, m.Y)
				break
			}
		}
	}
}

// missileExplode kills all enemies in blast radius and emits the explosion event.
func missileExplode(g *Game, x, y float64) {
	for i := range g.Enemies {
		e := &g.Enemies[i]
		if !e.Alive {
			continue
		}
		dx := x - e.X
		dy := y - e.Y
		if dx*dx+dy*dy < missileBlastRadSq {
			e.Alive = false
			g.Events = append(g.Events, Event{
				Type:  EventEnemyKilled,
				X:     e.X,
				Y:     e.Y,
				Value: float64(e.MaxHP) * 100,
			})
		}
	}

	g.Events = append(g.Events, Event{
		Type: EventMissileExploded, X: x, Y: y,
	})
}

func fireMissile(g *Game) {
	dx := math.Cos(g.Turret.Angle)
	dy := math.Sin(g.Turret.Angle)
	spawnX := g.Player.X + dx*TurretLength
	spawnY := g.Player.Y + dy*TurretLength

	g.Missiles = append(g.Missiles, Missile{
		X: spawnX, Y: spawnY,
		Angle: g.Turret.Angle,
		Speed: MissileInitialSpeed,
		Alive: true,
	})
	g.Events = append(g.Events, Event{
		Type: EventMissileFired, X: spawnX, Y: spawnY,
	})
}

// spawnMissileTrail emits smoke and flame particles behind the missile.
func spawnMissileTrail(g *Game, m *Missile) {
	tailX := m.X - math.Cos(m.Angle)*MissileRadius
	tailY := m.Y - math.Sin(m.Angle)*MissileRadius

	// Flame particles (bright, short-lived).
	for i := 0; i < 2; i++ {
		spread := (rand.Float64() - 0.5) * 0.6
		speed := 1.0 + rand.Float64()*1.5
		ejectAngle := m.Angle + math.Pi + spread
		life := 8 + rand.Intn(8)
		g.Particles = append(g.Particles, Particle{
			X: tailX, Y: tailY,
			VX:      math.Cos(ejectAngle) * speed,
			VY:      math.Sin(ejectAngle) * speed,
			Life:    life,
			MaxLife: life,
			Size:    2 + float32(rand.Float64()*2),
			Color:   colorMissileFlame,
		})
	}

	// Smoke particles (dim, longer-lived).
	if m.Age%2 == 0 {
		spread := (rand.Float64() - 0.5) * 0.4
		speed := 0.3 + rand.Float64()*0.8
		ejectAngle := m.Angle + math.Pi + spread
		life := 20 + rand.Intn(15)
		g.Particles = append(g.Particles, Particle{
			X: tailX, Y: tailY,
			VX:      math.Cos(ejectAngle) * speed,
			VY:      math.Sin(ejectAngle) * speed,
			Life:    life,
			MaxLife: life,
			Size:    3 + float32(rand.Float64()*2),
			Color:   colorMissileSmoke,
		})
	}
}

// emitBurst spawns a radial burst of particles — used by spawnMissileBlast layers.
func emitBurst(g *Game, x, y float64, count int, speedMin, speedMax float64, lifeMin, lifeMax int, sizeMin, sizeMax float32, col color.RGBA) {
	for i := 0; i < count; i++ {
		angle := rand.Float64() * 2 * math.Pi
		speed := speedMin + rand.Float64()*(speedMax-speedMin)
		life := lifeMin + rand.Intn(lifeMax-lifeMin+1)
		g.Particles = append(g.Particles, Particle{
			X: x, Y: y,
			VX:      math.Cos(angle) * speed,
			VY:      math.Sin(angle) * speed,
			Life:    life,
			MaxLife: life,
			Size:    sizeMin + float32(rand.Float64())*float32(sizeMax-sizeMin),
			Color:   col,
		})
	}
}

// spawnMissileBlast creates a large ring explosion for missile detonation.
func spawnMissileBlast(g *Game, x, y float64) {
	emitBurst(g, x, y, 25, 2.0, 7.0, 25, 44, 3, 7, colorBlastInner) // inner hot burst
	emitBurst(g, x, y, 20, 3.0, 7.0, 30, 54, 2, 5, colorBlastOuter) // outer ring
	emitBurst(g, x, y, 15, 0.5, 2.5, 40, 69, 4, 8, colorMissileSmoke) // smoke cloud
}

func drawMissiles(screen *ebiten.Image, g *Game, ox, oy float64) {
	for i := range g.Missiles {
		m := &g.Missiles[i]
		cx := float32(m.X + ox)
		cy := float32(m.Y + oy)
		cosA := float32(math.Cos(m.Angle))
		sinA := float32(math.Sin(m.Angle))

		// Pulsing outer glow.
		glowPulse := float32(1.0 + 0.2*math.Sin(float64(m.Age)*0.3))
		glowR := (MissileRadius + 4) * glowPulse
		glowCol := colorBlastOuter
		glowCol.A = 0x55
		vector.DrawFilledCircle(screen, cx, cy, glowR, glowCol, false)

		noseLen := float32(MissileRadius * 1.6)
		bodyW := float32(MissileRadius * 0.5)
		tailLen := float32(MissileRadius * 0.8)

		nx := cx + cosA*noseLen
		ny := cy + sinA*noseLen

		perpX := -sinA
		perpY := cosA
		lx := cx + perpX*bodyW
		ly := cy + perpY*bodyW
		rx := cx - perpX*bodyW
		ry := cy - perpY*bodyW

		tx := cx - cosA*tailLen
		ty := cy - sinA*tailLen

		// Nose cone.
		var nosePath vector.Path
		nosePath.MoveTo(nx, ny)
		nosePath.LineTo(lx, ly)
		nosePath.LineTo(rx, ry)
		nosePath.Close()
		vs, is := nosePath.AppendVerticesAndIndicesForFilling(nil, nil)
		for j := range vs {
			vs[j].ColorR = float32(colorMissileNose.R) / 255
			vs[j].ColorG = float32(colorMissileNose.G) / 255
			vs[j].ColorB = float32(colorMissileNose.B) / 255
			vs[j].ColorA = 1
		}
		screen.DrawTriangles(vs, is, emptyImage, nil)

		// Tail section.
		var tailPath vector.Path
		tailPath.MoveTo(lx, ly)
		tailPath.LineTo(tx, ty)
		tailPath.LineTo(rx, ry)
		tailPath.Close()
		vs, is = tailPath.AppendVerticesAndIndicesForFilling(nil, nil)
		for j := range vs {
			vs[j].ColorR = float32(colorMissileBody.R) / 255
			vs[j].ColorG = float32(colorMissileBody.G) / 255
			vs[j].ColorB = float32(colorMissileBody.B) / 255
			vs[j].ColorA = 1
		}
		screen.DrawTriangles(vs, is, emptyImage, nil)

		// Engine flame (flickering circles behind).
		flameOff := tailLen + 2
		for f := 0; f < 3; f++ {
			fSize := float32(2.0+rand.Float64()*3.0) * glowPulse
			fOff := flameOff + float32(f)*3
			fx := cx - cosA*fOff + float32(rand.Float64()-0.5)*2
			fy := cy - sinA*fOff + float32(rand.Float64()-0.5)*2
			col := colorMissileFlame
			if f == 0 {
				col = colorBlastInner
			}
			vector.DrawFilledCircle(screen, fx, fy, fSize, col, false)
		}
	}
}

// emptyImage is a 1x1 white pixel used for DrawTriangles.
var emptyImage = func() *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return img
}()
