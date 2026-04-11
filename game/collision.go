package game

const (
	projEnemyRadSq      = (ProjectileRadius + EnemyRadius) * (ProjectileRadius + EnemyRadius)
	playerEnemyRadSq    = (PlayerRadius + EnemyRadius) * (PlayerRadius + EnemyRadius)
	playerPowerUpRadSq  = (PlayerRadius + PowerUpRadius) * (PlayerRadius + PowerUpRadius)
	missileEnemyRadSq   = (MissileRadius + EnemyRadius) * (MissileRadius + EnemyRadius)
)

// aoeExplode kills all enemies within radiusSq of (x,y) and emits evt.
func aoeExplode(g *Game, x, y, radiusSq float32, evt EventType) {
	for i := range g.Enemies {
		e := &g.Enemies[i]
		if !e.Alive {
			continue
		}
		dx := x - e.X
		dy := y - e.Y
		if dx*dx+dy*dy < radiusSq {
			e.Alive = false
			g.Events = append(g.Events, Event{
				Type:   EventEnemyKilled,
				X:      e.X,
				Y:      e.Y,
				Value:  float32(e.MaxHP) * 100,
				Silent: true,
			})
		}
	}
	g.Events = append(g.Events, Event{Type: evt, X: x, Y: y})
}

// checkPowerUpCollisions detects player–powerup overlaps and emits pickup events.
// Separated so it can run during wave intro (when full collision checks are skipped).
func checkPowerUpCollisions(g *Game) {
	for i := range g.PowerUps {
		pu := &g.PowerUps[i]
		if pu.Life <= 0 {
			continue
		}
		dx := g.Player.X - pu.X
		dy := g.Player.Y - pu.Y
		if dx*dx+dy*dy < playerPowerUpRadSq {
			pu.Life = 0
			g.Events = append(g.Events, Event{
				Type:  EventPowerUpPickedUp,
				X:     pu.X,
				Y:     pu.Y,
				Value: float32(pu.Type),
			})
		}
	}
}

func checkCollisions(g *Game) {
	if !g.Player.Alive || g.State == StateRespawn {
		return
	}

	g.Player.CheckWalls(g)

	// Projectile vs enemy — squared distance, no sqrt.
	for i := range g.Projectiles {
		p := &g.Projectiles[i]
		if !p.Alive {
			continue
		}
		for j := range g.Enemies {
			e := &g.Enemies[j]
			if !e.Alive {
				continue
			}
			dx := p.X - e.X
			dy := p.Y - e.Y
			if dx*dx+dy*dy < projEnemyRadSq {
				p.Alive = false
				e.HP--
				if e.HP <= 0 {
					e.Alive = false
					g.Events = append(g.Events, Event{
						Type: EventEnemyKilled, X: e.X, Y: e.Y,
						Value: float32(e.MaxHP) * 100,
					})
				} else {
					e.FlashFrames = 6
					g.Events = append(g.Events, Event{
						Type: EventEnemyHit, X: e.X, Y: e.Y,
					})
				}
				break
			}
		}
	}

	checkPowerUpCollisions(g)


	// Enemy vs player — squared distance (skip if invulnerable).
	if g.Player.InvulnFrames > 0 {
		return
	}
	for i := range g.Enemies {
		e := &g.Enemies[i]
		if !e.Alive {
			continue
		}
		dx := g.Player.X - e.X
		dy := g.Player.Y - e.Y
		if dx*dx+dy*dy < playerEnemyRadSq {
			// Shield absorbs the hit: enemy dies, shield consumed.
			if g.PlayerPowerUps.Shield {
				g.PlayerPowerUps.Shield = false
				e.Alive = false
				g.Events = append(g.Events, Event{
					Type: EventShieldAbsorb, X: e.X, Y: e.Y,
				})
				g.Events = append(g.Events, Event{
					Type:  EventEnemyKilled,
					X:     e.X,
					Y:     e.Y,
					Value: float32(e.MaxHP) * 100,
				})
				break
			}
			g.Player.Alive = false
			g.Events = append(g.Events, Event{
				Type: EventPlayerDied, X: g.Player.X, Y: g.Player.Y,
			})
			break
		}
	}
}
