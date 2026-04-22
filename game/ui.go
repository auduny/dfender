package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Status bar layout.
const (
	statusTextY  = 22.0 // Y for FontHUD text in status bar
	statusScoreY = 16.0 // Y for FontMenu score (taller font)

	// Fixed X positions for power-up indicators (don't shift when others appear/disappear).
	puSlotShield = 200.0
	puSlotGuns   = 300.0
	puSlotCool   = 420.0
	puSlotMSL    = 540.0
	puSlotMine   = 640.0
)

func drawUI(screen *ebiten.Image, g *Game) {
	sbH := float32(ArenaMargin + StatusBarHeight - 5) // status bar background height

	// Status bar background.
	vector.DrawFilledRect(screen, 0, 0, float32(ScreenWidth), sbH,
		color.RGBA{0x05, 0x07, 0x14, 0xCC}, AntiAlias)

	// Wave indicator — left.
	waveStr := fmt.Sprintf("WAVE %d", g.Wave.Number)
	drawTextAt(screen, waveStr, FontHUD, 30, statusTextY, ColorBorder)

	// Score — center.
	scoreStr := fmt.Sprintf("%d", g.Score.Score)
	drawTextCentered(screen, scoreStr, FontMenu, statusScoreY, ColorUI)

	// Combo — right of score.
	if g.Score.Combo > 1 {
		comboStr := fmt.Sprintf("x%d", g.Score.Combo)
		sw, _ := text.Measure(scoreStr, FontMenu, 0)
		comboX := float64(ScreenWidth)/2 + sw/2 + 12
		drawTextAt(screen, comboStr, FontHUD, comboX, statusTextY+2, ColorBorder)
	}

	// Heat bar — right-center.
	drawHeatBar(screen, g)

	// Lives — far right.
	livesStr := fmt.Sprintf("LIVES: %d", g.Lives)
	drawTextAt(screen, livesStr, FontHUD, float64(ScreenWidth-130), statusTextY, ColorUI)

	// Power-up indicators — fixed positions.
	drawPowerUpHUD(screen, g)

	// Wave intro announcement — center screen overlay.
	if g.State == StateWaveIntro {
		waveAnnounce := fmt.Sprintf("WAVE %d", g.Wave.Number)
		drawTextCentered(screen, waveAnnounce, FontMenu, float64(ScreenHeight)/2-40, ColorBorder)
		if desc := waveDescription(g.Wave.Number); desc != "" {
			drawTextCentered(screen, desc, FontHUD, float64(ScreenHeight)/2, ColorUI)
		}
	}

	// Paused overlay with keymap.
	if g.State == StatePaused {
		if g.UnpauseTimer > 0 {
			drawTextCentered(screen, "RESUMING", FontMenu, float64(ScreenHeight)/2-40, ColorBorder)
		} else {
			drawPauseOverlay(screen, g)
		}
	}

	// Game over.
	if g.State == StateGameOver {
		cy := float64(ScreenHeight)/2 - 80
		drawTextCentered(screen, "GAME OVER", FontTitle, cy, color.RGBA{0xFF, 0x33, 0x33, 0xFF})
		cy += 90
		drawTextCentered(screen, fmt.Sprintf("SCORE: %d    WAVE: %d", g.Score.Score, g.Wave.Number), FontMenu, cy, ColorUI)
		cy += 50
		if g.HighScores.Qualifies(g.Score.Score) {
			drawTextCentered(screen, "NEW HIGH SCORE!", FontMenu, cy, ColorBorder)
			cy += 50
		}
		drawTextCentered(screen, "PRESS ENTER TO CONTINUE", FontMenuSmall, cy, ColorBorderDim)
	}

}

var keymapEntries = []struct{ key, action string }{
	{"WASD", "THRUST"},
	{"ARROWS", "AIM TURRET"},
	{"SPACE", "FIRE"},
	{"E", "LAUNCH MISSILE"},
	{"Q", "PLACE MINE"},
	{"M", "TOGGLE MUSIC"},
}

func drawPauseOverlay(screen *ebiten.Image, g *Game) {
	cy := float64(ScreenHeight)/2 - 160
	drawTextCentered(screen, "PAUSED", FontTitle, cy, ColorBorder)
	cy += 100

	keyX := float64(ScreenWidth)/2 - 140
	actX := float64(ScreenWidth)/2 + 20
	for _, k := range keymapEntries {
		action := k.action
		if k.key == "WASD" && g.Settings.CanonRelativeControls {
			action = "THRUST (CANON-REL)"
		}
		drawTextAt(screen, k.key, FontHUD, keyX, cy, ColorBorder)
		drawTextAt(screen, action, FontHUD, actX, cy, ColorUI)
		cy += 28
	}

	cy += 20
	drawTextCentered(screen, "PRESS P TO RESUME", FontMenuSmall, cy, ColorBorderDim)
}

func drawPowerUpHUD(screen *ebiten.Image, g *Game) {
	y := statusTextY

	if g.PlayerPowerUps.Shield {
		drawTextAt(screen, "SHIELD", FontHUD, puSlotShield, y, ColorBorder)
	}

	if g.PlayerPowerUps.GunsTimer > 0 {
		secs := g.PlayerPowerUps.GunsTimer / 60
		drawTextAt(screen, fmt.Sprintf("GUNS %ds", secs), FontHUD, puSlotGuns, y, ColorPlayer)
	}

	if g.PlayerPowerUps.SupercoolTimer > 0 {
		secs := g.PlayerPowerUps.SupercoolTimer / 60
		drawTextAt(screen, fmt.Sprintf("COOL %ds", secs), FontHUD, puSlotCool, y, ColorSupercool)
	}

	if g.PlayerPowerUps.MissileCount > 0 {
		drawTextAt(screen, fmt.Sprintf("MSL x%d", g.PlayerPowerUps.MissileCount), FontHUD, puSlotMSL, y, ColorHeatHot)
	}

	if g.PlayerPowerUps.MineCount > 0 {
		drawTextAt(screen, fmt.Sprintf("MINE x%d", g.PlayerPowerUps.MineCount), FontHUD, puSlotMine, y, ColorMine)
	}
}

func drawHeatBar(screen *ebiten.Image, g *Game) {
	barW := float32(180)
	barH := float32(10)
	barX := float32(ScreenWidth) - 340
	barY := float32(statusTextY + 5)

	// Background.
	vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{0x1A, 0x1A, 0x2E, 0xFF}, AntiAlias)

	// Fill.
	heat := g.Turret.Heat
	coolColor := ColorHeatCool
	if g.PlayerPowerUps.SupercoolTimer > 0 {
		coolColor = ColorSupercool
	}
	fillColor := lerpColor(coolColor, ColorHeatHot, heat)
	vector.DrawFilledRect(screen, barX, barY, barW*heat, barH, fillColor, AntiAlias)

	// Border.
	vector.StrokeRect(screen, barX, barY, barW, barH, 1, ColorBorderDim, AntiAlias)

}

// waveDescription returns an informational announcement for the given wave number.
func waveDescription(wave int) string {
	switch wave {
	case 1:
		return "DEFEND THE ARENA"
	case 2:
		return "ENEMIES HAVE 2 HP / DUAL GUNS DROP"
	case 3:
		return "ALPHA ENEMIES ARRIVE / FAST BUT UNWIELDY  / SUPERCOOL DROPS"
	case 4:
		return "ENEMIES HAVE 4 HP / MISSILE DROPS"
	case 5:
		return "BLINKY ENEMIES ARRIVE / THEY TELEPORT / MINE DROPS"
	case 6:
		return "ENEMIES HAVE 6 HP / GOOD LUCK"
	case 7:
		return "THE SWARM THICKENS"
	case 8:
		return "ENEMIES HAVE 8 HP"
	case 9:
		return "NO MERCY"
	case 10:
		return "CORNER DROPS ACTIVE / EXTRA LIFE DROPS / GOOD LUCK"
	default:
		return fmt.Sprintf("ENEMIES HAVE %d HP", wave)
	}
}
