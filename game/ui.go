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
	ebitenutil.DebugPrintAt(screen, scoreStr, ScreenWidth-150, 10)

	// Combo.
	if g.Score.Combo > 1 {
		comboStr := fmt.Sprintf("x%d", g.Score.Combo)
		ebitenutil.DebugPrintAt(screen, comboStr, ScreenWidth-150, 30)
	}

	// Wave — top center.
	if g.State == StateWaveIntro {
		waveStr := fmt.Sprintf("WAVE %d", g.Wave.Number)
		ebitenutil.DebugPrintAt(screen, waveStr, ScreenWidth/2-30, ScreenHeight/2-40)
	}

	// Lives — top right, below combo.
	livesStr := fmt.Sprintf("LIVES: %d", g.Lives)
	ebitenutil.DebugPrintAt(screen, livesStr, ScreenWidth-150, 50)

	// Heat bar — below lives.
	drawHeatBar(screen, g)

	// Game over.
	if g.State == StateGameOver {
		msg := fmt.Sprintf("GAME OVER\n\nSCORE: %d\nWAVE: %d\n\nPRESS ENTER TO RESTART", g.Score.Score, g.Wave.Number)
		ebitenutil.DebugPrintAt(screen, msg, ScreenWidth/2-80, ScreenHeight/2-40)
	}

	// FPS.
	ebitenutil.DebugPrintAt(screen, fmt.Sprintf("FPS: %.0f", ebiten.ActualFPS()), 10, 10)
}

func drawHeatBar(screen *ebiten.Image, g *Game) {
	barW := float32(150)
	barH := float32(8)
	barX := float32(ScreenWidth) - barW - 20
	barY := float32(70)

	// Background.
	vector.DrawFilledRect(screen, barX, barY, barW, barH, color.RGBA{0x1A, 0x1A, 0x2E, 0xFF}, false)

	// Fill.
	heat := float32(g.Turret.Heat)
	fillColor := lerpColor(ColorHeatCool, ColorHeatHot, heat)
	vector.DrawFilledRect(screen, barX, barY, barW*heat, barH, fillColor, false)

	// Border.
	vector.StrokeRect(screen, barX, barY, barW, barH, 1, ColorBorderDim, false)

	// Label if overheated.
	if g.Turret.Cooldown > 0 {
		ebitenutil.DebugPrintAt(screen, "OVERHEAT", int(barX)+75, int(barY)-16)
	}
}
