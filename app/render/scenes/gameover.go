package scenes

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type GameOverScene struct {
	loaded bool
	deps   *Deps
	// Add your scene-specific fields here
}

func NewGameOverScene(deps *Deps) *GameOverScene {
	return &GameOverScene{
		loaded: false,
		deps:   deps,
	}
}

func (s *GameOverScene) GetName() string {
	return "Game Over"
}

func (s *GameOverScene) Update() SceneId {
	// Handle escape key to exit
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ExitSceneId
	}

	// Handle input and update scene logic
	// Return the SceneId for the next scene or current scene

	return GameOverSceneId
}

func (s *GameOverScene) Draw(screen *ebiten.Image) {
	// Render your scene to the screen
	ebitenutil.DebugPrintAt(screen, "Hello World", 100, 100)
}

func (s *GameOverScene) FirstLoad() {
	// Initialize scene resources on first load
	s.loaded = true
}

func (s *GameOverScene) OnEnter() {
	// Called when transitioning to this scene
}

func (s *GameOverScene) OnExit() {
	// Called when leaving this scene
}

func (s *GameOverScene) IsLoaded() bool {
	return s.loaded
}

// Verify interface compliance
var _ Scene = (*GameOverScene)(nil)
