package game

// EventType identifies what happened.
type EventType int

const (
	EventEnemyKilled EventType = iota
	EventEnemyHit
	EventPlayerDied
	EventWallBounce
	EventWallDeath
	EventWaveComplete
	EventOverheat
	EventProjectileWallHit
	EventFired
	EventPowerUpPickedUp
	EventMissileFired
	EventMissileExploded
	EventShieldAbsorb
	EventEnemyWallDeath
)

// Event is a value type describing something that happened this frame.
type Event struct {
	Type   EventType
	X, Y   float64
	Value  float64
	Silent bool // suppress sound (e.g. missile blast covers individual kills)
}
