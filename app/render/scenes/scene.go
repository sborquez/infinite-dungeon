package scenes

import (
	"app/common"

	"github.com/hajimehoshi/ebiten/v2"
)

type SceneId uint

const (
	StartSceneId SceneId = iota
	BallsSceneId
	GravitySceneId
	ComfyUISceneId
	GameOverSceneId

	// Special scene
	ExitSceneId // Exit the game
)

type Scene interface {
	GetName() string
	Update() SceneId
	Draw(screen *ebiten.Image)
	FirstLoad()
	OnEnter()
	OnExit()
	IsLoaded() bool
}

type Deps struct {
	Config *common.Config
}

/*
Creating a New Scene - Template

Steps to create a new scene:
1. Add your scene ID to the SceneId constants above (e.g., MySceneId)
2. Copy the template below to a new file (e.g., myscene.go)
3. Replace "MyScene" with your actual scene name
4. Implement your scene logic

=== COPYABLE TEMPLATE START ===

package scenes

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type MyScene struct {
	loaded bool
	deps   *Deps
	// Add your scene-specific fields here
}

func NewMyScene(deps *Deps) *MyScene {
	return &MyScene{
		loaded: false,
		deps:   deps,
	}
}

func (s *MyScene) GetName() string {
	return "MyScene"
}

func (s *MyScene) Update() SceneId {
	// Handle escape key to exit
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		return ExitSceneId
	}

	// Handle input and update scene logic
	// Return the SceneId for the next scene or current scene

	return MySceneId
}

func (s *MyScene) Draw(screen *ebiten.Image) {
	// Render your scene to the screen
	// Example: ebitenutil.DebugPrintAt(screen, "Hello World", 100, 100)
}

func (s *MyScene) FirstLoad() {
	// Initialize scene resources on first load
	s.loaded = true
}

func (s *MyScene) OnEnter() {
	// Called when transitioning to this scene
}

func (s *MyScene) OnExit() {
	// Called when leaving this scene
}

func (s *MyScene) IsLoaded() bool {
	return s.loaded
}

// Verify interface compliance
var _ Scene = (*MyScene)(nil)

=== COPYABLE TEMPLATE END ===

See titlescene.go for a complete working example.
*/
