package game

import (
	"bytes"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

const (
	sampleRate   = 44100
	musicVolume  = 0.7
	maxConcurrent = 16  // Max simultaneous plays of the same SFX
	sfxVolume     = 0.3 // Per-voice volume to prevent clipping when many play at once
)

// SoundManager handles all audio: background music and sound effects.
type SoundManager struct {
	ctx         *audio.Context
	musicPlayer *audio.Player

	// Pre-generated SFX buffers
	sfxLaser          []byte
	sfxExplosion      []byte
	sfxSmallExplosion []byte
	sfxBounce         []byte
	sfxOverheat       []byte
	sfxWaveComplete   []byte
	sfxPlayerDeath    []byte
	sfxWallHit        []byte
	sfxPickup         []byte
	sfxShieldAbsorb   []byte
	sfxMissileLaunch     []byte
	sfxMissileExplode    []byte
	sfxOverheatWarning   []byte

	// Streaming thruster engine sound
	engine       *EngineSound
	enginePlayer *audio.Player

	// Track active players per SFX to limit polyphony
	activePlayers map[*[]byte][]*audio.Player
}

func NewSoundManager(musicData []byte) *SoundManager {
	ctx := audio.NewContext(sampleRate)

	sm := &SoundManager{
		ctx:               ctx,
		sfxLaser:          sfxLaser(),
		sfxExplosion:      sfxExplosion(),
		sfxSmallExplosion: sfxSmallExplosion(),
		sfxBounce:         sfxBounce(),
		sfxOverheat:       sfxOverheat(),
		sfxWaveComplete:   sfxWaveComplete(),
		sfxPlayerDeath:    sfxPlayerDeath(),
		sfxWallHit:        sfxWallHit(),
		sfxPickup:         sfxPickup(),
		sfxShieldAbsorb:   sfxShieldAbsorb(),
		sfxMissileLaunch:  sfxMissileLaunch(),
		sfxMissileExplode:    sfxMissileExplode(),
		sfxOverheatWarning:  sfxOverheatWarning(),
		engine:            NewEngineSound(),
		activePlayers:     make(map[*[]byte][]*audio.Player),
	}

	sm.initEngine()
	sm.initMusic(musicData)
	return sm
}

func (sm *SoundManager) initEngine() {
	p, err := sm.ctx.NewPlayer(sm.engine)
	if err != nil {
		log.Printf("sound: failed to create engine player: %v", err)
		return
	}
	p.SetVolume(0)
	p.Play()
	sm.enginePlayer = p
}

// SetThruster adjusts the engine drone based on how many thrusters are active (0-4).
func (sm *SoundManager) SetThruster(count int) {
	if sm.enginePlayer == nil {
		return
	}
	if count == 0 {
		sm.enginePlayer.SetVolume(0)
		return
	}
	// Shift fundamental higher with more thrusters: 110 → 180 Hz
	freq := 110.0 + float64(count-1)*23.0
	sm.engine.SetFreq(freq)
	sm.enginePlayer.SetVolume(0.5)
}

func (sm *SoundManager) initMusic(musicData []byte) {
	if len(musicData) == 0 {
		return
	}
	stream, err := mp3.DecodeWithoutResampling(bytes.NewReader(musicData))
	if err != nil {
		log.Printf("sound: failed to decode music: %v", err)
		return
	}

	loop := audio.NewInfiniteLoop(stream, stream.Length())
	player, err := sm.ctx.NewPlayer(loop)
	if err != nil {
		log.Printf("sound: failed to create music player: %v", err)
		return
	}

	player.SetVolume(musicVolume)
	sm.musicPlayer = player
}

// PlayMusic starts or resumes the background soundtrack.
func (sm *SoundManager) PlayMusic() {
	if sm.musicPlayer != nil && !sm.musicPlayer.IsPlaying() {
		sm.musicPlayer.Play()
	}
}

// PauseMusic pauses the background soundtrack.
func (sm *SoundManager) PauseMusic() {
	if sm.musicPlayer != nil && sm.musicPlayer.IsPlaying() {
		sm.musicPlayer.Pause()
	}
}

// ToggleMusic toggles the background soundtrack on/off.
func (sm *SoundManager) ToggleMusic() {
	if sm.musicPlayer == nil {
		return
	}
	if sm.musicPlayer.IsPlaying() {
		sm.musicPlayer.Pause()
	} else {
		sm.musicPlayer.Play()
	}
}

// play fires a one-shot SFX from a pre-generated buffer, limiting polyphony.
func (sm *SoundManager) play(buf *[]byte) {
	// Clean up finished players
	active := sm.activePlayers[buf]
	alive := active[:0]
	for _, p := range active {
		if p.IsPlaying() {
			alive = append(alive, p)
		}
	}
	sm.activePlayers[buf] = alive

	if len(alive) >= maxConcurrent {
		return
	}

	p := sm.ctx.NewPlayerFromBytes(*buf)
	p.SetVolume(sfxVolume)
	p.Play()
	sm.activePlayers[buf] = append(sm.activePlayers[buf], p)
}

// HandleEvent plays the appropriate sound for a game event.
func (sm *SoundManager) HandleEvent(e Event) {
	switch e.Type {
	case EventEnemyKilled, EventEnemyWallDeath:
		sm.play(&sm.sfxExplosion)
	case EventEnemyHit:
		sm.play(&sm.sfxSmallExplosion)
	case EventPlayerDied:
		sm.play(&sm.sfxPlayerDeath)
	case EventWallBounce:
		sm.play(&sm.sfxBounce)
	case EventWallDeath:
		sm.play(&sm.sfxPlayerDeath)
	case EventWaveComplete:
		sm.play(&sm.sfxWaveComplete)
	case EventOverheat:
		sm.play(&sm.sfxOverheat)
	case EventOverheatWarning:
		sm.play(&sm.sfxOverheatWarning)
	case EventProjectileWallHit:
		sm.play(&sm.sfxWallHit)
	case EventFired:
		sm.play(&sm.sfxLaser)
	case EventPowerUpPickedUp:
		sm.play(&sm.sfxPickup)
	case EventShieldAbsorb:
		sm.play(&sm.sfxShieldAbsorb)
	case EventMissileFired:
		sm.play(&sm.sfxMissileLaunch)
	case EventMissileExploded:
		sm.play(&sm.sfxMissileExplode)
	}
}
