package scenes

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	log "github.com/sirupsen/logrus"
)

type StartScene struct {
	loaded bool
	deps   *Deps

	// Scene selector
	selectedScene int
	scenes        []SceneOption
}

type SceneOption struct {
	id   SceneId
	name string
}

func NewStartScene(deps *Deps) *StartScene {
	log.Info("Creating new start scene (title screen)")

	// Build from deps.Scenes
	scenesOptions := []SceneOption{
		{id: BallsSceneId, name: "Balls Physics Demo"},
		{id: GravitySceneId, name: "Gravity Demo"},
		{id: ComfyUISceneId, name: "ComfyUI Demo"},
		{id: GameOverSceneId, name: "Game Over"},
	}

	log.WithField("available_scenes", len(scenesOptions)).Info("Built scene options from dependencies")
	for i, option := range scenesOptions {
		log.WithFields(log.Fields{
			"index":      i,
			"scene_id":   option.id,
			"scene_name": option.name,
		}).Debug("Added scene option")
	}

	return &StartScene{
		loaded:        false,
		deps:          deps,
		selectedScene: 0,
		scenes:        scenesOptions,
	}
}

func (s *StartScene) GetName() string {
	return "Main Menu"
}

func (s *StartScene) Draw(screen *ebiten.Image) {
	width := float32(s.deps.Config.Render.Window.Width)
	height := float32(s.deps.Config.Render.Window.Height)

	log.WithFields(log.Fields{
		"screen_width":   width,
		"screen_height":  height,
		"scene_count":    len(s.scenes),
		"selected_index": s.selectedScene,
	}).Trace("Drawing title scene")

	// Draw gradient background
	s.drawGradientBackground(screen, width, height)

	// Draw title with shadow and background
	title := "Infinite Dungeon"
	titleX := width / 2
	titleY := height * 0.18
	titleFontSize := 36
	titleBoxPadding := 16
	textW := len(title) * titleFontSize / 2
	boxW := float32(textW) + float32(titleBoxPadding*2)
	boxH := float32(titleFontSize) + float32(titleBoxPadding*2)
	boxX := titleX - boxW/2
	boxY := titleY - float32(titleBoxPadding)
	// Draw semi-transparent box
	vector.DrawFilledRect(screen, boxX, boxY, boxW, boxH, color.RGBA{0, 0, 0, 180}, false)
	// Draw shadow
	ebitenutil.DebugPrintAt(screen, title, int(titleX)-textW/22, int(titleY)+2)
	// Draw title in white
	ebitenutil.DebugPrintAt(screen, title, int(titleX)-textW/2, int(titleY))

	// Draw scene selector with background
	selectorY := height * 0.4
	selectorBoxW := width * 0.6
	selectorBoxH := float32(len(s.scenes)*48 + 32)
	selectorBoxX := width/2 - selectorBoxW/2
	selectorBoxY := selectorY - 24
	vector.DrawFilledRect(screen, selectorBoxX, selectorBoxY, selectorBoxW, selectorBoxH, color.RGBA{0, 0, 0, 160}, false)
	s.drawSceneSelector(screen, width, height, selectorY)

	// Draw instructions with background
	instructions := "Use ↑↓ arrows to select, ENTER to start"
	instructionsX := width / 2
	instructionsY := height * 0.8
	instrBoxW := float32(len(instructions)*12 + 32)
	instrBoxH := float32(32)
	instrBoxX := instructionsX - instrBoxW/2
	instrBoxY := instructionsY - 8
	vector.DrawFilledRect(screen, instrBoxX, instrBoxY, instrBoxW, instrBoxH, color.RGBA{0, 0, 0, 160}, false)
	ebitenutil.DebugPrintAt(screen, instructions, int(instructionsX)-len(instructions)*6, int(instructionsY))
}

func (s *StartScene) drawGradientBackground(screen *ebiten.Image, width, height float32) {
	topColor := color.RGBA{203, 0, 5, 255}     // Dark blue
	bottomColor := color.RGBA{80, 20, 10, 255} // Purple

	// Draw gradient by creating multiple rectangles
	numSteps := 50
	for i := 0; i < numSteps; i++ {
		y1 := float32(i) * height / float32(numSteps)
		y2 := float32(i+1) * height / float32(numSteps)

		// Interpolate colors
		t := float32(i) / float32(numSteps-1)
		r := uint8(float32(topColor.R)*(1-t) + float32(bottomColor.R)*t)
		g := uint8(float32(topColor.G)*(1-t) + float32(bottomColor.G)*t)
		b := uint8(float32(topColor.B)*(1-t) + float32(bottomColor.B)*t)

		gradientColor := color.RGBA{r, g, b, 255}
		vector.DrawFilledRect(screen, 0, y1, width, y2-y1, gradientColor, false)
	}
}

func (s *StartScene) drawSceneSelector(screen *ebiten.Image, width, height, startY float32) {
	log.WithFields(log.Fields{
		"scene_count":      len(s.scenes),
		"selected_index":   s.selectedScene,
		"start_y_position": startY,
	}).Trace("Drawing scene selector menu")

	spacing := 48
	for i, scene := range s.scenes {
		y := startY + float32(i)*float32(spacing)
		x := width / 2

		// Draw selection indicator
		if i == s.selectedScene {
			log.WithFields(log.Fields{
				"selected_scene":    scene.name,
				"selected_scene_id": scene.id,
				"menu_position":     i,
			}).Trace("Highlighting selected menu item")

			// Highlight selected item
			ebitenutil.DebugPrintAt(screen, ">", int(x)-120, int(y))
			ebitenutil.DebugPrintAt(screen, "<", int(x)+len(scene.name)*12+8, int(y))
		}
		// Draw scene name
		// (col variable removed, not used)
		ebitenutil.DebugPrintAt(screen, scene.name, int(x)-len(scene.name)*6, int(y))
	}
}

func (s *StartScene) FirstLoad() {
	log.WithField("scene", "StartScene").Info("First load of title scene")
	s.loaded = true
}

func (s *StartScene) IsLoaded() bool {
	return s.loaded
}

func (s *StartScene) OnEnter() {
	log.WithFields(log.Fields{
		"scene":            "StartScene",
		"available_scenes": len(s.scenes),
		"selected_index":   s.selectedScene,
	}).Info("Entered title scene")

	if len(s.scenes) > 0 && s.selectedScene < len(s.scenes) {
		log.WithField("selected_scene", s.scenes[s.selectedScene].name).Debug("Default scene selection")
	}
}

func (s *StartScene) OnExit() {
	log.WithField("scene", "StartScene").Info("Exiting title scene")
}

func (s *StartScene) Update() SceneId {
	// Safety check to prevent divide by zero
	if len(s.scenes) == 0 {
		log.Warn("No scenes available in title menu, staying on start scene")
		return StartSceneId
	}

	// Ensure selectedScene is within bounds
	if s.selectedScene >= len(s.scenes) {
		log.WithFields(log.Fields{
			"current_index": s.selectedScene,
			"scene_count":   len(s.scenes),
		}).Warn("Selected scene index out of bounds, resetting to 0")
		s.selectedScene = 0
	}

	// Handle scene selection with arrow keys
	if inpututil.IsKeyJustPressed(ebiten.KeyUp) {
		previousSelection := s.selectedScene
		s.selectedScene = (s.selectedScene - 1 + len(s.scenes)) % len(s.scenes)
		log.WithFields(log.Fields{
			"direction":      "up",
			"previous_index": previousSelection,
			"new_index":      s.selectedScene,
			"selected_scene": s.scenes[s.selectedScene].name,
		}).Debug("Scene selection changed")
	}

	if inpututil.IsKeyJustPressed(ebiten.KeyDown) {
		previousSelection := s.selectedScene
		s.selectedScene = (s.selectedScene + 1) % len(s.scenes)
		log.WithFields(log.Fields{
			"direction":      "down",
			"previous_index": previousSelection,
			"new_index":      s.selectedScene,
			"selected_scene": s.scenes[s.selectedScene].name,
		}).Debug("Scene selection changed")
	}

	// Handle scene selection with enter
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if len(s.scenes) > 0 {
			selectedScene := s.scenes[s.selectedScene]
			log.WithFields(log.Fields{
				"selected_index":      s.selectedScene,
				"selected_scene_id":   selectedScene.id,
				"selected_scene_name": selectedScene.name,
			}).Info("User selected scene, transitioning")
			return selectedScene.id
		} else {
			log.Warn("Enter pressed but no scenes available")
		}
	}

	return StartSceneId
}

var _ Scene = (*StartScene)(nil)
