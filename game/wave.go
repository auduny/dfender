package game

import "math/rand"

const (
	SpawnInterval = 20 // frames between each enemy spawn within a wave
)

type WaveManager struct {
	Number     int
	IntroTick  int
	SpawnQueue int // enemies left to spawn this wave
	SpawnTimer int // frames until next spawn
	GateIndex  int // round-robin gate selection
}

func NewWaveManager() WaveManager {
	return WaveManager{Number: 1}
}

func (w *WaveManager) NextWave() {
	w.Number++
	w.IntroTick = 0
}

func (w *WaveManager) StartSpawning(g *Game) {
	// Enemies per wave: 6 + 2*wave
	w.SpawnQueue = 6 + 2*w.Number
	w.SpawnTimer = 0
}

func (w *WaveManager) Update(g *Game) {
	// Spawn enemies from queue.
	if w.SpawnQueue > 0 {
		w.SpawnTimer--
		if w.SpawnTimer <= 0 {
			w.SpawnTimer = SpawnInterval
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

	speed := EnemyBaseSpeed + float64(w.Number-1)*0.2
	hp := w.Number
	turnRate := EnemyTurnRateMin + rand.Float64()*(EnemyTurnRateMax-EnemyTurnRateMin)

	g.Enemies = append(g.Enemies, Enemy{
		X: gate.X, Y: gate.Y,
		VX: gate.DirX * speed, VY: gate.DirY * speed,
		Speed:    speed,
		TurnRate: turnRate,
		HP: hp, MaxHP: hp,
		Alive: true,
	})
}
