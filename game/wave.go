package game

import "math/rand"

const (
	SpawnWindowSeconds = 2.0 // all enemies in a wave spawn within this time
)

type WaveManager struct {
	Number     int
	IntroTick  int
	SpawnQueue    int // enemies left to spawn this wave
	SpawnTimer    int // frames until next spawn
	SpawnInterval int // frames between spawns (computed per wave)
	GateIndex     int // round-robin gate selection
}

func NewWaveManager() WaveManager {
	return WaveManager{Number: 1}
}

func (w *WaveManager) NextWave() {
	w.Number++
	w.IntroTick = 0
}

func (w *WaveManager) StartSpawning(g *Game) {
	// Enemies per wave: 6 + 2*wave, capped at wave 5 count (16).
	count := 6 + 2*w.Number
	if count > 16 {
		count = 16
	}
	w.SpawnQueue = count
	// Distribute spawns evenly across the spawn window.
	w.SpawnTimer = 0
	w.SpawnInterval = int(SpawnWindowSeconds * 60 / float64(count))
}

func (w *WaveManager) Update(g *Game) {
	// Spawn enemies from queue.
	if w.SpawnQueue > 0 {
		w.SpawnTimer--
		if w.SpawnTimer <= 0 {
			w.SpawnTimer = w.SpawnInterval
			w.spawnOne(g)
			w.SpawnQueue--
		}
	}

	// Check wave complete: no enemies left and queue empty.
	if w.SpawnQueue == 0 && len(g.Enemies) == 0 && g.State == StatePlaying {
		g.Events = append(g.Events, Event{Type: EventWaveComplete, Value: float64(w.Number)})
		g.Score.AddWaveBonus(w.Number)
	}
}

func (w *WaveManager) spawnOne(g *Game) {
	gates := Gates()
	gate := gates[w.GateIndex%4]
	w.GateIndex++

	hp := w.Number

	// Pick enemy type based on wave number.
	// Wave 1-2: all Normal. Wave 3+: mix in Reds. Wave 5+: mix in Greens.
	eType := EnemyNormal
	if w.Number >= 3 {
		roll := rand.Float64()
		if w.Number >= 5 && roll < 0.25 {
			eType = EnemyGreen
		} else if roll < 0.5 {
			eType = EnemyRed
		}
	}

	var speed, turnRate, accel, maxSpeed float64
	var evadeSign float64

	switch eType {
	case EnemyNormal:
		speed = EnemyNormalSpeed + float64(w.Number-1)*0.15
		turnRate = EnemyNormalTurnRate
	case EnemyRed:
		speed = EnemyRedBaseSpeed
		turnRate = EnemyRedTurnRate
		accel = EnemyRedAccel
		maxSpeed = EnemyRedMaxSpeed + float64(w.Number-1)*0.3
	case EnemyGreen:
		speed = EnemyGreenSpeed + float64(w.Number-1)*0.1
		turnRate = EnemyGreenTurnRate
		if rand.Intn(2) == 0 {
			evadeSign = 1
		} else {
			evadeSign = -1
		}
	}

	teleportTimer := 0
	if eType == EnemyGreen {
		teleportTimer = EnemyGreenTeleportMin + rand.Intn(EnemyGreenTeleportMax-EnemyGreenTeleportMin)
	}

	g.Enemies = append(g.Enemies, Enemy{
		X: gate.X, Y: gate.Y,
		VX: gate.DirX * speed, VY: gate.DirY * speed,
		Speed:         speed,
		TurnRate:      turnRate,
		HP:            hp,
		MaxHP:         hp,
		Alive:         true,
		Type:          eType,
		Accel:         accel,
		MaxSpeed:      maxSpeed,
		EvadeSign:     evadeSign,
		TeleportTimer: teleportTimer,
	})
}
