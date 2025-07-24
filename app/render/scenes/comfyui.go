package scenes

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type ComfyUIScene struct {
	loaded bool
	deps   *Deps
	// Add your scene-specific fields here
}

func NewComfyUIScene(deps *Deps) *ComfyUIScene {
	return &ComfyUIScene{
		loaded: false,
		deps:   deps,
	}
}

func (s *ComfyUIScene) GetName() string {
	return "ComfyUI Demo"
}

func (s *ComfyUIScene) Update() SceneId {
	// Handle escape key to exit
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ExitSceneId
	}

	// Handle input and update scene logic
	// Return the SceneId for the next scene or current scene

	return ComfyUISceneId
}

func (s *ComfyUIScene) Draw(screen *ebiten.Image) {
	// Render your scene to the screen
	// Example: ebitenutil.DebugPrintAt(screen, "Hello World", 100, 100)
}

func (s *ComfyUIScene) FirstLoad() {
	// Initialize scene resources on first load
	s.loaded = true
}

func (s *ComfyUIScene) OnEnter() {
	// Called when transitioning to this scene
}

func (s *ComfyUIScene) OnExit() {
	// Called when leaving this scene
}

func (s *ComfyUIScene) IsLoaded() bool {
	return s.loaded
}

// Verify interface compliance
var _ Scene = (*ComfyUIScene)(nil)
