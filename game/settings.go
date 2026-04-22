package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

func (g *Game) updateSettings() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		g.State = StateMenu
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyEnter) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) ||
		inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		g.Settings.CanonRelativeControls = !g.Settings.CanonRelativeControls
	}
}

func (g *Game) drawSettings(screen *ebiten.Image) {
	cy := float64(ScreenHeight)/2 - 120

	drawTextCentered(screen, "SETTINGS", FontMenu, cy, ColorBorder)
	cy += 90

	drawTextCentered(screen, "CANON-RELATIVE STEERING", FontMenuSmall, cy, ColorUI)
	cy += 40

	var valStr string
	valColor := ColorUI
	if g.Settings.CanonRelativeControls {
		valStr = "[ ON ]"
		valColor = ColorBorder
	} else {
		valStr = "[ OFF ]"
	}
	drawTextCentered(screen, valStr, FontMenu, cy, valColor)
	cy += 60

	drawTextCentered(screen, "SPACE / ENTER / ARROWS TO TOGGLE", FontMenuSmall, cy, ColorBorderDim)
	cy += 36
	drawTextCentered(screen, "ESC TO RETURN", FontMenuSmall, cy, ColorBorderDim)
}
