package game

import (
	"fmt"
	"image/color"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const menuItemCount = 4

var menuLabels = [menuItemCount]string{"NEW GAME", "HIGH SCORES", "CREDITS", "QUIT"}

func (g *Game) updateMenu() {
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeyS) {
		g.MenuSelection = (g.MenuSelection + 1) % menuItemCount
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyW) {
		g.MenuSelection = (g.MenuSelection + menuItemCount - 1) % menuItemCount
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		switch g.MenuSelection {
		case 0: // New Game
			g.reset()
		case 1: // High Scores
			g.State = StateHighScores
		case 2: // Credits
			g.State = StateCredits
		case 3: // Quit
			os.Exit(0)
		}
	}
}

func (g *Game) updateCredits() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.State = StateMenu
	}
}

func (g *Game) updateHighScores() {
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		g.State = StateMenu
	}
}

// drawTextCentered draws text horizontally centered at the given Y position.
func drawTextCentered(screen *ebiten.Image, s string, face *text.GoTextFace, y float64, clr color.Color) {
	w, _ := text.Measure(s, face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(ScreenWidth)/2-w/2, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, s, face, op)
}

// drawTextAt draws text at the given position.
func drawTextAt(screen *ebiten.Image, s string, face *text.GoTextFace, x, y float64, clr color.Color) {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(screen, s, face, op)
}

func (g *Game) drawMenuScreen(screen *ebiten.Image) {
	s := g.Shaders

	// Starfield background → scene → bloom.
	s.SceneImage.Clear()
	s.DrawStarfield(s.SceneImage, g.Tick)

	// Draw showcase enemies and powerups into scene (before bloom).
	if g.State == StateMenu {
		drawMenuEnemies(s.SceneImage, g.Tick)
		drawMenuPowerUps(s.SceneImage, g.Tick)
	}

	// Bloom the scene.
	s.ApplyBloom(screen)

	// Everything below is post-bloom (crisp, no blur artifacts).

	// Decorative border frame.
	borderInset := float32(60)
	bw := float32(ScreenWidth) - borderInset*2
	bh := float32(ScreenHeight) - borderInset*2
	vector.StrokeRect(screen, borderInset, borderInset, bw, bh, 2, ColorBorder, AntiAlias)

	// Corner accents.
	accent := float32(20)
	vector.StrokeLine(screen, borderInset, borderInset+accent, borderInset+accent, borderInset, 2, ColorBorder, AntiAlias)
	vector.StrokeLine(screen, float32(ScreenWidth)-borderInset-accent, borderInset, float32(ScreenWidth)-borderInset, borderInset+accent, 2, ColorBorder, AntiAlias)
	vector.StrokeLine(screen, borderInset, float32(ScreenHeight)-borderInset-accent, borderInset+accent, float32(ScreenHeight)-borderInset, 2, ColorBorder, AntiAlias)
	vector.StrokeLine(screen, float32(ScreenWidth)-borderInset-accent, float32(ScreenHeight)-borderInset, float32(ScreenWidth)-borderInset, float32(ScreenHeight)-borderInset-accent, 2, ColorBorder, AntiAlias)

	// Labels.
	if g.State == StateMenu {
		drawMenuEnemyLabels(screen)
		drawMenuPowerUpLabels(screen, g.Tick)
	}

	// Title.
	drawTextCentered(screen, "dFENDER", FontTitle, 180, ColorBorder)

	// Subtitle.
	drawTextCentered(screen, "ARENA SHOOTER", FontMenu, 270, ColorBorderDim)

	if g.State == StateCredits {
		g.drawCredits(screen)
		return
	}

	if g.State == StateHighScores {
		g.drawHighScores(screen)
		return
	}

	// Menu items.
	menuStartY := float64(ScreenHeight)/2 - 40
	for i, label := range menuLabels {
		y := menuStartY + float64(i)*50

		clr := color.Color(ColorUI)
		if i == g.MenuSelection {
			clr = ColorBorder
			label = "> " + label + " <"
		}
		drawTextCentered(screen, label, FontMenu, y, clr)
	}

}

type menuEnemy struct {
	label    string
	innerCol color.RGBA
	cx       float32
	labelW   float64 // cached text width
}

var menuEnemies []menuEnemy

func initMenuEnemies() {
	menuEnemies = []menuEnemy{
		{"JOE", ColorEnemyNormal, float32(ScreenWidth)/2 - 200, 0},
		{"ALPHA", ColorEnemyRed, float32(ScreenWidth) / 2, 0},
		{"BLINKY", ColorEnemyGreen, float32(ScreenWidth)/2 + 200, 0},
	}
	for i := range menuEnemies {
		menuEnemies[i].labelW, _ = text.Measure(menuEnemies[i].label, FontMenuSmall, 0)
	}
}

const (
	menuEnemyCY       = float32(ScreenHeight) - 300
	menuEnemyR        = float32(EnemyRadius) * menuShowcaseScale
	menuPowerUpCY     = float32(ScreenHeight) - 180
	menuPowerUpR      = float32(PowerUpRadius) * menuShowcaseScale
	menuShowcaseScale = 1.3
)

// drawMenuEnemies draws one of each enemy type on the menu screen as decoration.
func drawMenuEnemies(screen *ebiten.Image, tick uint64) {
	if menuEnemies == nil {
		initMenuEnemies()
	}

	angle := float32(tick) * 0.02
	for _, item := range menuEnemies {
		drawEnemyShape(screen, item.cx, menuEnemyCY, menuEnemyR, angle, ColorEnemy, item.innerCol)
	}
}

// drawMenuEnemyLabels draws enemy labels (post-bloom so text stays crisp).
func drawMenuEnemyLabels(screen *ebiten.Image) {
	if menuEnemies == nil {
		return
	}

	for _, item := range menuEnemies {
		drawTextAt(screen, item.label, FontMenuSmall,
			float64(item.cx)-item.labelW/2, float64(menuEnemyCY+menuEnemyR+16), item.innerCol)
	}
}

type menuPowerUp struct {
	label  string
	col    color.RGBA
	sides  int
	cx     float32
	labelW float64
}

var menuPowerUps []menuPowerUp

func initMenuPowerUps() {
	menuPowerUps = []menuPowerUp{
		{"SHIELD", ColorShield, 6, 0, 0},
		{"GUNS", ColorPlayer, 5, 0, 0},
		{"MISSILE", ColorHeatHot, 4, 0, 0},
		{"COOL", ColorSupercool, 7, 0, 0},
		{"MINE", ColorMine, -5, 0, 0},
		{"LIFE", ColorExtraLife, 0, 0, 0},
	}
	spacing := float32(200)
	startX := float32(ScreenWidth)/2 - spacing*float32(len(menuPowerUps)-1)/2
	for i := range menuPowerUps {
		menuPowerUps[i].cx = startX + float32(i)*spacing
		menuPowerUps[i].labelW, _ = text.Measure(menuPowerUps[i].label, FontMenuSmall, 0)
	}
}

// drawMenuPowerUps draws powerup shapes (for bloomed scene).
func drawMenuPowerUps(screen *ebiten.Image, tick uint64) {
	if menuPowerUps == nil {
		initMenuPowerUps()
	}

	angle := float32(tick) * PowerUpRotSpeed
	bob := sin32(float32(tick)*PowerUpBobSpeed) * PowerUpBobAmount

	for _, item := range menuPowerUps {
		py := menuPowerUpCY + bob
		vector.StrokeCircle(screen, item.cx, py, menuPowerUpR+4, 2, item.col, AntiAlias)
		if item.sides > 0 {
			drawPolygon(screen, item.cx, py, menuPowerUpR, item.sides, angle, 3, item.col)
		} else if item.sides < 0 {
			drawStar(screen, item.cx, py, menuPowerUpR, -item.sides, angle, 3, item.col)
		} else {
			drawHeart(screen, item.cx, py, menuPowerUpR, angle, 3, item.col)
		}
		vector.DrawFilledCircle(screen, item.cx, py, 4, item.col, AntiAlias)
	}
}

// drawMenuPowerUpLabels draws powerup labels (post-bloom, so text stays crisp).
func drawMenuPowerUpLabels(screen *ebiten.Image, tick uint64) {
	if menuPowerUps == nil {
		return
	}

	bob := sin32(float32(tick)*PowerUpBobSpeed) * PowerUpBobAmount

	for _, item := range menuPowerUps {
		py := menuPowerUpCY + bob
		drawTextAt(screen, item.label, FontMenuSmall,
			float64(item.cx)-item.labelW/2, float64(py+menuPowerUpR+16), item.col)
	}
}

func (g *Game) drawCredits(screen *ebiten.Image) {
	cy := float64(ScreenHeight)/2 - 100

	drawTextCentered(screen, "CREDITS", FontMenu, cy, ColorBorder)
	cy += 60

	drawTextCentered(screen, "Game Design. Programming. Music.", FontMenuSmall, cy, ColorUI)
	cy += 30
	drawTextCentered(screen, "Per Buer", FontMenu, cy, ColorBorder)
	cy += 60

	drawTextCentered(screen, "AI Code Generation", FontMenuSmall, cy, ColorUI)
	cy += 30
	drawTextCentered(screen, "Claude Opus 4.6", FontMenu, cy, ColorBorder)
	cy += 60

	drawTextCentered(screen, "Built with Ebitengine in Go", FontMenuSmall, cy, ColorUI)
	cy += 80

	drawTextCentered(screen, "https://github.com/perbu/dfender", FontMenuSmall, cy, ColorUI)
	cy += 100

	drawTextCentered(screen, "PRESS ENTER OR ESC TO RETURN", FontMenuSmall, cy, ColorBorderDim)
}

func (g *Game) drawHighScores(screen *ebiten.Image) {
	cy := float64(ScreenHeight)/2 - 200

	drawTextCentered(screen, "HIGH SCORES", FontMenu, cy, ColorBorder)
	cy += 60

	if len(g.HighScores.Entries) == 0 {
		drawTextCentered(screen, "No scores yet. Go play!", FontMenuSmall, cy+40, ColorUI)
	} else {
		// Header.
		header := fmt.Sprintf("%-4s  %-12s  %8s  %5s   %s", "#", "NAME", "SCORE", "WAVE", "DATE")
		drawTextCentered(screen, header, FontMenuSmall, cy, ColorBorderDim)
		cy += 35

		for i, e := range g.HighScores.Entries {
			line := fmt.Sprintf("%-4d  %-12s  %8d  %5d   %s",
				i+1, e.Name, e.Score, e.Wave, e.Date.Format("2006-01-02"))
			clr := color.Color(ColorUI)
			if i == 0 {
				clr = ColorBorder
			}
			drawTextCentered(screen, line, FontMenuSmall, cy, clr)
			cy += 30
		}
	}

	cy = float64(ScreenHeight)/2 + 200
	drawTextCentered(screen, "PRESS ENTER OR ESC TO RETURN", FontMenuSmall, cy, ColorBorderDim)
}
