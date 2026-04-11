package game

import (
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	ScreenWidth  = 1920
	ScreenHeight = 1080

	// Arena inset from screen edges.
	ArenaMargin = 40

	// Extra top margin for the status bar HUD.
	StatusBarHeight = 30

	// Gate dimensions.
	GateWidth = 120
)

// Art deco color palette.
var (
	ColorBackground = color.RGBA{0x0A, 0x0E, 0x27, 0xFF}
	ColorBorder     = color.RGBA{0xD4, 0xA8, 0x43, 0xFF}
	ColorBorderDim  = color.RGBA{0x6A, 0x54, 0x22, 0xFF}
	ColorPlayer     = color.RGBA{0xF5, 0xD6, 0x7B, 0xFF}
	ColorEnemy      = color.RGBA{0x00, 0xC9, 0xA7, 0xFF}
	ColorEnemyHurt  = color.RGBA{0xFF, 0x69, 0xB4, 0xFF} // hot pink at 0% HP
	ColorProjectile = color.RGBA{0xFF, 0xE0, 0x80, 0xFF}
	ColorUI         = color.RGBA{0xF0, 0xE6, 0xD3, 0xFF}
	ColorHeatCool   = color.RGBA{0xD4, 0xA8, 0x43, 0xFF}
	ColorHeatHot    = color.RGBA{0xFF, 0x33, 0x33, 0xFF}
	ColorShield     = color.RGBA{0x33, 0xDD, 0x55, 0xFF} // green for shield powerup
	ColorSupercool  = color.RGBA{0x44, 0xBB, 0xFF, 0xFF} // blue for supercooling powerup
	ColorMine       = color.RGBA{0xFF, 0x99, 0x00, 0xFF} // orange for mine powerup
	ColorSmoke      = color.RGBA{0x99, 0x77, 0x55, 0xCC} // brownish smoke (explosions/trails)

	// Enemy inner colors (by type).
	ColorEnemyNormal = color.RGBA{0x66, 0x66, 0x77, 0xFF} // muted gray (won't blow out under bloom)
	ColorEnemyRed    = color.RGBA{0xFF, 0x22, 0x22, 0xFF} // saturated red
	ColorEnemyGreen  = color.RGBA{0x22, 0xFF, 0x22, 0xFF} // saturated green
)

// GameState tracks what phase the game is in.
type GameState int

const (
	StateMenu GameState = iota
	StatePlaying
	StateGameOver
	StateWaveIntro
	StateCredits
	StateHighScores
	StateRespawn
	StatePaused
)

const (
	StartingLives  = 3
	RespawnFreeze  = 180 // 3 seconds at 60 TPS
	UnpauseFreeze  = 60  // 1 second at 60 TPS
)

// Game is the top-level state container.
type Game struct {
	State GameState

	// Player
	Player Player

	// Turret
	Turret Turret

	// Projectiles (pooled slice, reused)
	Projectiles []Projectile

	// Enemies
	Enemies []Enemy

	// Particles
	Particles []Particle

	// Wave
	Wave WaveManager

	// Score
	Score ScoreTracker

	// Power-ups
	PowerUps       []PowerUp
	Missiles       []Missile
	Mines          []Mine
	PlayerPowerUps PlayerPowerUps

	// Events (cleared each frame)
	Events []Event

	// Lives
	Lives        int
	RespawnTimer int // frames remaining in respawn freeze

	// Screen shake
	ShakeFrames int
	ShakeAmount float32

	// Tick counter for animations
	Tick uint64

	// Shaders and render pipeline
	Shaders *Shaders

	// Sound
	Sound *SoundManager

	// Menu
	MenuSelection int // 0=New Game, 1=High Scores, 2=Credits, 3=Quit

	// High scores
	HighScores HighScoreTable

	// Pause
	UnpauseTimer   int       // frames remaining in unpause freeze (0 = not unpausing)
	PrePauseState  GameState // state to return to after unpause

	iconSet bool // true after window icon has been generated
}

func New(musicData, fontData []byte) *Game {
	InitFonts(fontData)
	g := &Game{
		State:      StateMenu,
		Shaders:    NewShaders(),
		Sound:      NewSoundManager(musicData),
		HighScores: LoadHighScores(),
	}
	g.Sound.PlayMusic()
	return g
}

func (g *Game) reset() {
	g.State = StateWaveIntro
	g.Player = NewPlayer(ScreenWidth/2, ScreenHeight/2+100)
	g.Turret = NewTurret()
	g.Projectiles = g.Projectiles[:0]
	g.Enemies = g.Enemies[:0]
	g.Particles = g.Particles[:0]
	if cap(g.Events) < 32 {
		g.Events = make([]Event, 0, 32)
	} else {
		g.Events = g.Events[:0]
	}
	g.Wave = NewWaveManager()
	g.Score = ScoreTracker{}
	g.PowerUps = g.PowerUps[:0]
	g.Missiles = g.Missiles[:0]
	g.Mines = g.Mines[:0]
	g.PlayerPowerUps = PlayerPowerUps{}
	g.Lives = StartingLives
	g.RespawnTimer = 0
	g.ShakeFrames = 0
	g.Tick = 0
}

func (g *Game) Update() error {
	g.Tick++

	if !g.iconSet {
		g.iconSet = true
		g.generateWindowIcon()
	}

	// Global key: toggle music.
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.Sound.ToggleMusic()
	}

	// Tick powerup timers (shared across playing states).
	if g.PlayerPowerUps.GunsTimer > 0 {
		g.PlayerPowerUps.GunsTimer--
	}
	if g.PlayerPowerUps.SupercoolTimer > 0 {
		g.PlayerPowerUps.SupercoolTimer--
	}

	switch g.State {
	case StateMenu:
		g.updateMenu()
	case StateCredits:
		g.updateCredits()
	case StateHighScores:
		g.updateHighScores()
	case StatePlaying:
		if inpututil.IsKeyJustPressed(ebiten.KeyP) {
			g.Sound.SetThruster(0)
			g.PrePauseState = StatePlaying
			g.State = StatePaused
			break
		}
		g.Events = g.Events[:0]
		g.updatePlaying()
	case StatePaused:
		g.updatePaused()
	case StateWaveIntro:
		if inpututil.IsKeyJustPressed(ebiten.KeyP) {
			g.Sound.SetThruster(0)
			g.PrePauseState = StateWaveIntro
			g.State = StatePaused
			break
		}
		g.Events = g.Events[:0]
		g.updateWaveIntro()
		// Drain events so juice still plays during wave intro.
		for _, e := range g.Events {
			applyJuice(g, e)

			switch e.Type {
			case EventPowerUpPickedUp:
				g.applyPowerUp(e)
			}
		}
	case StateRespawn:
		g.Sound.SetThruster(0)
		g.updateRespawn()
	case StateGameOver:
		g.Sound.SetThruster(0)
		// Decay screen shake (post-mortem).
		if g.ShakeFrames > 0 {
			g.ShakeFrames--
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.submitHighScore()
			g.State = StateMenu
			g.MenuSelection = 0
		}
	}
	return nil
}

func (g *Game) updatePlaying() {
	// Systems — order matters.
	g.Player.Update()
	g.Player.SpawnThrustParticles(g)
	g.Sound.SetThruster(g.Player.ThrusterCount())
	g.Turret.Update(g)
	updateProjectiles(g)
	updateEnemies(g)
	spawnEnemyThrustParticles(g)
	updatePowerUps(g)
	updateMissiles(g)
	updateMines(g)
	checkCollisions(g)
	checkMissileCollisions(g)
	checkMineCollisions(g)
	g.Score.Update()
	g.Wave.Update(g)
	updateParticles(g)

	// Drain events.
	for _, e := range g.Events {
		applyJuice(g, e)

		switch e.Type {
		case EventEnemyKilled:
			g.Score.AddKill(int(e.Value))
			spawnPowerUpDrop(g, e.X, e.Y, g.Wave.Number)
		case EventPlayerDied, EventWallDeath:
			g.Lives--
			g.Score.Combo = 0
			g.Score.ComboTimer = 0
			if g.Lives <= 0 {
				g.State = StateGameOver
			} else {
				g.respawn()
			}
			return
		case EventWaveComplete:
			g.State = StateWaveIntro
			g.Wave.NextWave()
			return
		case EventPowerUpPickedUp:
			g.applyPowerUp(e)
		}
	}

	// Decay screen shake.
	if g.ShakeFrames > 0 {
		g.ShakeFrames--
	}
}

func (g *Game) applyPowerUp(e Event) {
	puType := PowerUpType(int(e.Value))
	switch puType {
	case PowerUpShield:
		g.PlayerPowerUps.Shield = true
	case PowerUpGuns:
		g.PlayerPowerUps.GunsTimer = GunsBuffDuration
	case PowerUpMissile:
		if g.PlayerPowerUps.MissileCount < MissileMaxCount {
			g.PlayerPowerUps.MissileCount++
		}
	case PowerUpSupercool:
		g.PlayerPowerUps.SupercoolTimer = SupercoolBuffDuration
	case PowerUpMine:
		if g.PlayerPowerUps.MineCount < MineMaxCount {
			g.PlayerPowerUps.MineCount++
		}
	}
}

func (g *Game) updateWaveIntro() {
	// Keep the game alive during wave intro — player can still move, particles animate, etc.
	g.Player.Update()
	g.Player.CheckWalls(g)
	g.Player.SpawnThrustParticles(g)
	g.Sound.SetThruster(g.Player.ThrusterCount())
	g.Turret.Update(g)
	updateProjectiles(g)
	updatePowerUps(g)
	checkPowerUpCollisions(g)
	updateParticles(g)

	g.Wave.IntroTick++
	if g.Wave.IntroTick > 210 { // 3.5 seconds at 60fps
		g.State = StatePlaying
		g.Wave.StartSpawning(g)
	}
}

func (g *Game) submitHighScore() {
	if g.HighScores.Qualifies(g.Score.Score) {
		g.HighScores.Add(HighScoreEntry{
			Name:  getUsername(),
			Score: g.Score.Score,
			Wave:  g.Wave.Number,
			Date:  time.Now(),
		})
		g.HighScores.Save()
	}
}

func (g *Game) respawn() {
	g.Player = NewPlayer(ScreenWidth/2, ScreenHeight/2+100)
	g.Turret.Heat = 0
	g.Turret.Cooldown = 0
	g.RespawnTimer = RespawnFreeze
	g.State = StateRespawn
}

func (g *Game) updateRespawn() {
	// Particles still animate, but enemies are frozen.
	updateParticles(g)

	// Decay screen shake.
	if g.ShakeFrames > 0 {
		g.ShakeFrames--
	}

	g.RespawnTimer--
	if g.RespawnTimer <= 0 {
		g.Player.InvulnFrames = InvulnDuration
		g.State = StatePlaying
	}
}

func (g *Game) updatePaused() {
	if g.UnpauseTimer > 0 {
		// Counting down to resume — animate particles so the screen isn't fully frozen.
		updateParticles(g)
		g.UnpauseTimer--
		if g.UnpauseTimer <= 0 {
			g.State = g.PrePauseState
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyP) {
		g.UnpauseTimer = UnpauseFreeze
	}
}

func (g *Game) Draw(screen *ebiten.Image) {
	s := g.Shaders

	if g.State == StateMenu || g.State == StateCredits || g.State == StateHighScores {
		g.drawMenuScreen(screen)
		return
	}

	// Screen shake offset.
	var ox, oy float32
	if g.ShakeFrames > 0 {
		ox = sin32(float32(g.Tick)*13.7) * g.ShakeAmount
		oy = cos32(float32(g.Tick)*17.3) * g.ShakeAmount
	}

	// --- Render scene to offscreen image ---
	s.SceneImage.Clear()

	// Shader background (art deco grid).
	s.DrawBackground(s.SceneImage, g.Tick)

	// Gate portals (shader swirl).
	spawning := g.Wave.SpawnQueue > 0
	for _, gate := range Gates() {
		s.DrawGatePortal(s.SceneImage, gate, g.Tick, spawning)
	}

	// Game objects.
	drawParticles(s.SceneImage, g, ox, oy)
	drawPowerUps(s.SceneImage, g, ox, oy)
	drawProjectiles(s.SceneImage, g, ox, oy)
	drawMines(s.SceneImage, g, ox, oy)
	drawMissiles(s.SceneImage, g, ox, oy)
	drawEnemies(s.SceneImage, g, ox, oy)
	g.Player.Draw(s.SceneImage, ox, oy, g.Turret.Heat)
	drawShieldOverlay(s.SceneImage, g, ox, oy)
	g.Turret.Draw(s.SceneImage, g, ox, oy)

	// --- Post-processing ---

	// Heat distortion (when gun is hot).
	turretTipX := g.Player.X + cos32(g.Turret.Angle)*TurretLength
	turretTipY := g.Player.Y + sin32(g.Turret.Angle)*TurretLength
	if g.Turret.Heat > 0.3 {
		s.HeatTemp.Clear()
		s.ApplyHeatDistortion(s.HeatTemp, s.SceneImage, g.Turret.Heat, turretTipX, turretTipY, g.Tick)
		s.SceneImage.Clear()
		s.SceneImage.DrawImage(s.HeatTemp, nil)
	}

	// Bloom post-process → screen.
	s.ApplyBloom(screen)

	// Arena border (post-bloom, stays crisp).
	drawArena(screen, g, ox, oy)

	// UI drawn on top (not bloomed).
	drawUI(screen, g)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return ScreenWidth, ScreenHeight
}

// Arena bounds (inside the margin).
func ArenaLeft() float32   { return float32(ArenaMargin) }
func ArenaRight() float32  { return float32(ScreenWidth - ArenaMargin) }
func ArenaTop() float32    { return float32(ArenaMargin + StatusBarHeight) }
func ArenaBottom() float32 { return float32(ScreenHeight - ArenaMargin) }

// Gate centers.
type Gate struct {
	X, Y    float32
	DirX    float32 // spawn direction
	DirY    float32
}

var gates = [4]Gate{
	{float32(ScreenWidth) / 2, ArenaTop(), 0, 1},
	{float32(ScreenWidth) / 2, ArenaBottom(), 0, -1},
	{ArenaLeft(), float32(ScreenHeight) / 2, 1, 0},
	{ArenaRight(), float32(ScreenHeight) / 2, -1, 0},
}

func Gates() [4]Gate { return gates }

func drawArena(screen *ebiten.Image, g *Game, ox, oy float32) {
	l := ArenaLeft() + ox
	r := ArenaRight() + ox
	t := ArenaTop() + oy
	b := ArenaBottom() + oy
	w := float32(3.0)

	gates := Gates()
	hw := float32(GateWidth) / 2

	// Draw border segments, leaving gaps for gates.
	// Top edge (north gate)
	gc := gates[0].X + ox
	vector.StrokeLine(screen, l, t, gc-hw, t, w, ColorBorder, AntiAlias)
	vector.StrokeLine(screen, gc+hw, t, r, t, w, ColorBorder, AntiAlias)

	// Bottom edge (south gate)
	gc = gates[1].X + ox
	vector.StrokeLine(screen, l, b, gc-hw, b, w, ColorBorder, AntiAlias)
	vector.StrokeLine(screen, gc+hw, b, r, b, w, ColorBorder, AntiAlias)

	// Left edge (west gate)
	gc = gates[2].Y + oy
	vector.StrokeLine(screen, l, t, l, gc-hw, w, ColorBorder, AntiAlias)
	vector.StrokeLine(screen, l, gc+hw, l, b, w, ColorBorder, AntiAlias)

	// Right edge (east gate)
	gc = gates[3].Y + oy
	vector.StrokeLine(screen, r, t, r, gc-hw, w, ColorBorder, AntiAlias)
	vector.StrokeLine(screen, r, gc+hw, r, b, w, ColorBorder, AntiAlias)

	// Gate markers are now drawn by the gate shader.
}
