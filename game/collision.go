package game

const (
	projEnemyRadSq   = (ProjectileRadius + EnemyRadius) * (ProjectileRadius + EnemyRadius)
	playerEnemyRadSq = (PlayerRadius + EnemyRadius) * (PlayerRadius + EnemyRadius)
)

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
						Value: float64(e.MaxHP) * 100,
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
			g.Player.Alive = false
			g.Events = append(g.Events, Event{
				Type: EventPlayerDied, X: g.Player.X, Y: g.Player.Y,
			})
			break
		}
	}
}
