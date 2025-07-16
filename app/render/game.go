package render

import (
	"app/common"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// Game implements ebiten.Game interface
type Game struct {
	Config *common.Config
}

func NewGame(config *common.Config) *Game {
	return &Game{Config: config}
}

func RunGame(game *Game) {
	ebiten.RunGame(game)
}

func (g *Game) Update() error {
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrint(screen, "Hello Go")
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return 320, 240
}
