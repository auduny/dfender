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
	colorMissileSmoke = ColorSmoke
	colorBlastInner   = color.RGBA{0xFF, 0xFF, 0xDD, 0xFF} // white-hot center
	colorBlastOuter   = color.RGBA{0xFF, 0x66, 0x00, 0xFF} // orange ring
)

type Missile struct {
	X, Y  float32
	Angle float32
	Speed float32
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
		headX := cos32(m.Angle)
		headY := sin32(m.Angle)
		bestDist := float32(math.MaxFloat32)
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
				bestAngle = atan232(dy, dx)
				hasTarget = true
			}
		}

		// Steer toward target.
		if hasTarget {
			diff := remainder32(bestAngle-m.Angle, 2*pi32)
			if diff > MissileTurnRate {
				diff = MissileTurnRate
			} else if diff < -MissileTurnRate {
				diff = -MissileTurnRate
			}
			m.Angle += diff
		}

		vx := cos32(m.Angle) * m.Speed
		vy := sin32(m.Angle) * m.Speed
		m.X += vx
		m.Y += vy

		// Spawn smoke trail particles every frame.
		spawnMissileTrail(g, m)

		// Wall collision — explode on impact.
		if m.X < ArenaLeft() || m.X > ArenaRight() ||
			m.Y < ArenaTop() || m.Y > ArenaBottom() {
			ix := max(ArenaLeft(), min(m.X, ArenaRight()))
			iy := max(ArenaTop(), min(m.Y, ArenaBottom()))
			m.Alive = false
			missileExplode(g, ix, iy)
		}
	}

	g.Missiles = compact(g.Missiles, func(m *Missile) bool { return m.Alive })
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

func missileExplode(g *Game, x, y float32) {
	aoeExplode(g, x, y, missileBlastRadSq, EventMissileExploded)
}

func fireMissile(g *Game) {
	dx := cos32(g.Turret.Angle)
	dy := sin32(g.Turret.Angle)
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
	tailX := m.X - cos32(m.Angle)*MissileRadius
	tailY := m.Y - sin32(m.Angle)*MissileRadius

	// Flame particles (bright, short-lived).
	for i := 0; i < 2; i++ {
		spread := (rand.Float32() - 0.5) * 0.6
		speed := 1.0 + rand.Float32()*1.5
		ejectAngle := m.Angle + pi32 + spread
		life := 8 + rand.Intn(8)
		g.Particles = append(g.Particles, Particle{
			X: tailX, Y: tailY,
			VX:      cos32(ejectAngle) * speed,
			VY:      sin32(ejectAngle) * speed,
			Life:    life,
			MaxLife: life,
			Size:    2 + rand.Float32()*2,
			Color:   colorMissileFlame,
		})
	}

	// Smoke particles (dim, longer-lived).
	if m.Age%2 == 0 {
		spread := (rand.Float32() - 0.5) * 0.4
		speed := 0.3 + rand.Float32()*0.8
		ejectAngle := m.Angle + pi32 + spread
		life := 20 + rand.Intn(15)
		g.Particles = append(g.Particles, Particle{
			X: tailX, Y: tailY,
			VX:      cos32(ejectAngle) * speed,
			VY:      sin32(ejectAngle) * speed,
			Life:    life,
			MaxLife: life,
			Size:    3 + rand.Float32()*2,
			Color:   colorMissileSmoke,
		})
	}
}

// emitBurst spawns a radial burst of particles — used by spawnMissileBlast layers.
func emitBurst(g *Game, x, y float32, count int, speedMin, speedMax float32, lifeMin, lifeMax int, sizeMin, sizeMax float32, col color.RGBA) {
	for i := 0; i < count; i++ {
		angle := rand.Float32() * 2 * pi32
		speed := speedMin + rand.Float32()*(speedMax-speedMin)
		life := lifeMin + rand.Intn(lifeMax-lifeMin+1)
		g.Particles = append(g.Particles, Particle{
			X: x, Y: y,
			VX:      cos32(angle) * speed,
			VY:      sin32(angle) * speed,
			Life:    life,
			MaxLife: life,
			Size:    sizeMin + rand.Float32()*(sizeMax-sizeMin),
			Color:   col,
		})
	}
}

// spawnMissileBlast creates a large ring explosion for missile detonation.
func spawnMissileBlast(g *Game, x, y float32) {
	emitBurst(g, x, y, 25, 2.0, 7.0, 25, 44, 3, 7, colorBlastInner) // inner hot burst
	emitBurst(g, x, y, 20, 3.0, 7.0, 30, 54, 2, 5, colorBlastOuter) // outer ring
	emitBurst(g, x, y, 15, 0.5, 2.5, 40, 69, 4, 8, colorMissileSmoke) // smoke cloud
}

func drawMissiles(screen *ebiten.Image, g *Game, ox, oy float32) {
	for i := range g.Missiles {
		m := &g.Missiles[i]
		cx := m.X + ox
		cy := m.Y + oy
		cosA := cos32(m.Angle)
		sinA := sin32(m.Angle)

		// Pulsing outer glow.
		glowPulse := 1.0 + 0.2*sin32(float32(m.Age)*0.3)
		glowR := (MissileRadius + 4) * glowPulse
		glowCol := colorBlastOuter
		glowCol.A = 0x55
		vector.DrawFilledCircle(screen, cx, cy, glowR, glowCol, AntiAlias)

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
			vector.DrawFilledCircle(screen, fx, fy, fSize, col, AntiAlias)
		}
	}
}

// emptyImage is a 1x1 white pixel used for DrawTriangles.
var emptyImage = func() *ebiten.Image {
	img := ebiten.NewImage(1, 1)
	img.Fill(color.White)
	return img
}()
