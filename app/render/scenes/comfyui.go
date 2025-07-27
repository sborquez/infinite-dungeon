package scenes

import (
	"app/services"
	"math"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type ComfyUIScene struct {
	loaded bool
	deps   *Deps

	// Add your scene-specific fields here
	backgroundImage *ebiten.Image
}

func NewComfyUIScene(deps *Deps) *ComfyUIScene {
	log.Info("Creating new ComfyUI scene")

	if deps == nil {
		log.Error("ComfyUI scene dependencies are nil")
	} else if deps.ComfyUI == nil {
		log.Error("ComfyUI service is nil in dependencies")
	} else {
		log.WithField("comfyui_running", deps.ComfyUI.IsRunning()).Debug("ComfyUI service status")
	}

	scene := &ComfyUIScene{
		loaded:          false,
		deps:            deps,
		backgroundImage: nil,
	}

	log.WithField("scene_address", &scene).Debug("ComfyUI scene created successfully")
	return scene
}

func (s *ComfyUIScene) GetName() string {
	return "ComfyUI Demo"
}

func (s *ComfyUIScene) Update() SceneId {
	// Handle escape key to exit
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		log.Debug("Escape key pressed in ComfyUI scene, exiting")
		return ExitSceneId
	}

	// Handle input and update scene logic
	// Return the SceneId for the next scene or current scene

	return ComfyUISceneId
}

func (s *ComfyUIScene) Draw(screen *ebiten.Image) {
	// Render your scene to the screen
	// Rotate and zoom in and out the background image given the current time
	if s.backgroundImage != nil {
		log.Trace("Drawing ComfyUI background image with transformations")

		options := &ebiten.DrawImageOptions{}

		// Apply rotation using GeoM (geometry matrix)
		bounds := s.backgroundImage.Bounds()
		centerX, centerY := float64(bounds.Dx()/2), float64(bounds.Dy()/2)

		timeValue := float64(time.Now().UnixNano()) / 1e9
		scaleValue := 1.5 + math.Sin(timeValue)

		log.WithFields(log.Fields{
			"image_width":  bounds.Dx(),
			"image_height": bounds.Dy(),
			"center_x":     centerX,
			"center_y":     centerY,
			"rotation":     timeValue,
			"scale":        scaleValue,
		}).Trace("Image transformation parameters")

		options.GeoM.Translate(-centerX, -centerY)
		options.GeoM.Rotate(timeValue)
		options.GeoM.Scale(scaleValue, scaleValue)
		options.GeoM.Translate(centerX, centerY)

		// Translate the image to the center of the screen
		screenCenterX := float64(screen.Bounds().Dx() / 2)
		screenCenterY := float64(screen.Bounds().Dy() / 2)
		options.GeoM.Translate(screenCenterX, screenCenterY)

		screen.DrawImage(s.backgroundImage, options)
	} else {
		log.Trace("No background image available, drawing placeholder")
		// Draw placeholder text when no image is available
		// Could add ebitenutil.DebugPrintAt here if needed
		ebitenutil.DebugPrintAt(screen, "No background image available", 10, 10)
	}
}

func (s *ComfyUIScene) FirstLoad() {
	log.Info("Starting first load of ComfyUI scene")

	// Initialize scene resources on first load
	s.loaded = true

	// Check dependencies before making ComfyUI request
	if s.deps == nil {
		log.Error("Cannot load ComfyUI scene: dependencies are nil")
		return
	}

	if s.deps.ComfyUI == nil {
		log.Error("Cannot load ComfyUI scene: ComfyUI service is nil")
		return
	}

	if !s.deps.ComfyUI.IsRunning() {
		log.Warn("ComfyUI service is not running, attempting to start")
		if err := s.deps.ComfyUI.Start(); err != nil {
			log.WithError(err).Error("Failed to start ComfyUI service")
			return
		}
	}

	log.Info("Requesting image from ComfyUI service")

	// Create image request with proper parameters
	imageRequest := services.ImageRequest{
		WorkflowName:  "default.json",
		ContentPrompt: "A beautiful space station in the sky, seen from the ground",
		StylePrompt:   "Unrealistic, 3D render, pixel, starcraft 1 artstyle",
		Seed:          42,
		Steps:         20,
		Ratio:         services.ImageRatioLandscape,
	}

	log.WithFields(log.Fields{
		"workflow":       imageRequest.WorkflowName,
		"content_prompt": imageRequest.ContentPrompt,
		"style_prompt":   imageRequest.StylePrompt,
		"seed":           imageRequest.Seed,
		"steps":          imageRequest.Steps,
		"ratio":          imageRequest.Ratio,
	}).Debug("ComfyUI image request parameters")

	results, err := s.deps.ComfyUI.NewImageFromPrompt(imageRequest)
	if err != nil {
		log.WithError(err).Error("Failed to get image from ComfyUI")
		return
	}

	if results == nil {
		log.Error("ComfyUI returned nil results")
		return
	}

	if results.Image == nil {
		log.Error("ComfyUI returned nil image")
		return
	}

	s.backgroundImage = results.Image
	bounds := s.backgroundImage.Bounds()

	log.WithFields(log.Fields{
		"image_width":  bounds.Dx(),
		"image_height": bounds.Dy(),
		"image_bounds": bounds,
	}).Info("Successfully loaded ComfyUI background image")
}

func (s *ComfyUIScene) OnEnter() {
	log.WithField("scene", "ComfyUI").Info("Entering ComfyUI scene")

	if s.backgroundImage != nil {
		bounds := s.backgroundImage.Bounds()
		log.WithFields(log.Fields{
			"has_background": true,
			"image_size":     bounds,
		}).Debug("ComfyUI scene entered with background image")
	} else {
		log.WithField("has_background", false).Debug("ComfyUI scene entered without background image")
	}
}

func (s *ComfyUIScene) OnExit() {
	log.WithField("scene", "ComfyUI").Info("Exiting ComfyUI scene")

	if s.backgroundImage != nil {
		log.Debug("Deallocating ComfyUI background image")
		s.backgroundImage.Deallocate()
		s.backgroundImage = nil
		s.loaded = false
		log.Debug("ComfyUI background image deallocated successfully")
	} else {
		log.Debug("No background image to deallocate")
	}
}

func (s *ComfyUIScene) IsLoaded() bool {
	loaded := s.loaded
	log.WithField("is_loaded", loaded).Trace("ComfyUI scene load status checked")
	return loaded
}

// Verify interface compliance
var _ Scene = (*ComfyUIScene)(nil)
