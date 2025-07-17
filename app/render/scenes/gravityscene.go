package scenes

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

type GravityScene struct {
	loaded bool
	deps   *Deps
}

func NewGravityScene(deps *Deps) *GravityScene {
	return &GravityScene{
		loaded: false,
		deps:   deps,
	}
}

func (s *GravityScene) Draw(screen *ebiten.Image) {
	ebitenutil.DebugPrintAt(screen, "Gravity Demo (press Q to return)", 40, 40)
	fps := ebiten.ActualFPS()
	fpsStr := fmt.Sprintf("FPS: %0.1f", fps)
	ebitenutil.DebugPrintAt(screen, fpsStr, 10, 10)
}

func (s *GravityScene) FirstLoad() {
	s.loaded = true
}

func (s *GravityScene) IsLoaded() bool {
	return s.loaded
}

func (s *GravityScene) OnEnter() {}
func (s *GravityScene) OnExit()  {}

func (s *GravityScene) Update() SceneId {
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		return StartSceneId
	}
	return GravitySceneId
}

var _ Scene = (*GravityScene)(nil)
