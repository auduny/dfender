package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

func drawUI(screen *ebiten.Image, g *Game) {
	// Score — top right.
	scoreStr := fmt.Sprintf("%d", g.Score.Score)
	drawTextAt(screen, scoreStr, FontHUD, float64(ScreenWidth-150), 10, ColorUI)

	// Combo.
	if g.Score.Combo > 1 {
		comboStr := fmt.Sprintf("x%d", g.Score.Combo)
		drawTextAt(screen, comboStr, FontHUD, float64(ScreenWidth-150), 34, ColorBorder)
	}

	// Wave — center screen announcement.
	if g.State == StateWaveIntro {
		waveStr := fmt.Sprintf("WAVE %d", g.Wave.Number)
		drawTextCentered(screen, waveStr, FontMenu, float64(ScreenHeight)/2-40, ColorBorder)
	}

	// Lives — top right, below combo.
	livesStr := fmt.Sprintf("LIVES: %d", g.Lives)
	drawTextAt(screen, livesStr, FontHUD, float64(ScreenWidth-150), 56, ColorUI)

	// Heat bar — below lives.
	drawHeatBar(screen, g)

	// Power-up indicators — below heat bar.
	drawPowerUpHUD(screen, g)

	// Paused overlay.
	if g.State == StatePaused {
		if g.UnpauseTimer > 0 {
			drawTextCentered(screen, "RESUMING", FontMenu, float64(ScreenHeight)/2-40, ColorBorder)
		} else {
			drawTextCentered(screen, "PAUSED", FontTitle, float64(ScreenHeight)/2-80, ColorBorder)
			drawTextCentered(screen, "PRESS P TO RESUME", FontMenuSmall, float64(ScreenHeight)/2, ColorBorderDim)
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

	// FPS (keep debug font — it's a debug readout).
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("FPS: %.0f", ebiten.ActualFPS()), 10, 10)
}

func drawPowerUpHUD(screen *ebiten.Image, g *Game) {
	baseX := float64(ScreenWidth - 170)
	baseY := float64(95)

	if g.PlayerPowerUps.Shield {
		drawTextAt(screen, "SHIELD", FontHUD, baseX, baseY, ColorBorder)
		baseY += 20
	}

	if g.PlayerPowerUps.GunsTimer > 0 {
		secs := g.PlayerPowerUps.GunsTimer / 60
		drawTextAt(screen, fmt.Sprintf("GUNS %ds", secs), FontHUD, baseX, baseY, ColorPlayer)
		baseY += 20
	}

	if g.PlayerPowerUps.SupercoolTimer > 0 {
		secs := g.PlayerPowerUps.SupercoolTimer / 60
		drawTextAt(screen, fmt.Sprintf("COOL %ds", secs), FontHUD, baseX, baseY, ColorSupercool)
		baseY += 20
	}

	if g.PlayerPowerUps.MissileCount > 0 {
		drawTextAt(screen, fmt.Sprintf("MSL x%d [E]", g.PlayerPowerUps.MissileCount), FontHUD, baseX, baseY, ColorHeatHot)
	}
}

func drawHeatBar(screen *ebiten.Image, g *Game) {
	barW := float32(150)
	barH := float32(8)
	barX := float32(ScreenWidth) - barW - 20
	barY := float32(78)

	// Background.
	vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{0x1A, 0x1A, 0x2E, 0xFF}, AntiAlias)

	// Fill.
	heat := float32(g.Turret.Heat)
	coolColor := ColorHeatCool
	if g.PlayerPowerUps.SupercoolTimer > 0 {
		coolColor = ColorSupercool
	}
	fillColor := lerpColor(coolColor, ColorHeatHot, heat)
	vector.DrawFilledRect(screen, barX, barY, barW*heat, barH, fillColor, AntiAlias)

	// Border.
	vector.StrokeRect(screen, barX, barY, barW, barH, 1, ColorBorderDim, AntiAlias)

	// Label if overheated — to the left of the bar so it doesn't overlap lives.
	if g.Turret.Cooldown > 0 {
		drawTextAt(screen, "OVERHEAT", FontHUD, float64(barX)-90, float64(barY)-2, ColorHeatHot)
	}
}
