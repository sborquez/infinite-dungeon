package render

import (
	"app/common"

	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/hajimehoshi/ebiten/v2"

	"app/render/scenes"
)

// Game implements ebiten.Game interface

const (
	WINDOW_TITLE = "Infinite Dungeon"
)

type Game struct {
	Config        *common.Config
	Width, Height int

	availableScenes map[scenes.SceneId]scenes.Scene
	activeSceneId   scenes.SceneId
	shutdown        bool
}

func NewGame(config *common.Config) *Game {
	log.Info("Initializing new game instance")

	w := config.Render.Window.Width
	h := config.Render.Window.Height
	log.WithFields(log.Fields{
		"width":      w,
		"height":     h,
		"fullscreen": config.Render.Window.Fullscreen,
	}).Info("Setting up game window configuration")

	// Initialize shared dependencies
	deps := &scenes.Deps{
		Config: config,
	}
	log.Debug("Initialized scene dependencies")

	// Populate deps.Scenes with the game scenes
	availableScenes := map[scenes.SceneId]scenes.Scene{
		scenes.StartSceneId:    scenes.NewStartScene(deps),
		scenes.BallsSceneId:    scenes.NewBallsScene(deps),
		scenes.GravitySceneId:  scenes.NewGravityScene(deps),
		scenes.ComfyUISceneId:  scenes.NewComfyUIScene(deps),
		scenes.GameOverSceneId: scenes.NewGameOverScene(deps),
	}

	activeSceneId := scenes.StartSceneId
	log.WithField("initial_scene", activeSceneId).Info("Setting initial active scene")

	availableScenes[activeSceneId].FirstLoad()
	log.WithField("scene_id", activeSceneId).Debug("Initial scene loaded")

	log.Info("Game initialization complete")
	return &Game{
		Config:          config,
		availableScenes: availableScenes,
		activeSceneId:   activeSceneId,
		shutdown:        false,

		Width:  w,
		Height: h,
	}
}

func StopGame(g *Game) error {
	log.WithField("active_scene", g.activeSceneId).Info("Shutting down Game")

	// Stop current scene
	activeScene := g.availableScenes[g.activeSceneId]
	log.WithField("scene_id", g.activeSceneId).Debug("Calling OnExit for active scene")
	activeScene.OnExit()

	log.Info("Game shutdown complete")
	// Exit
	return ebiten.Termination
}

func RunGame(g *Game) {
	log.WithField("title", WINDOW_TITLE).Info("Starting game")

	ebiten.SetWindowTitle(WINDOW_TITLE)
	log.WithField("title", WINDOW_TITLE).Debug("Set window title")

	// Set window size BEFORE setting fullscreen
	ebiten.SetWindowSize(g.Width, g.Height)
	log.WithFields(log.Fields{
		"width":  g.Width,
		"height": g.Height,
	}).Debug("Set window size")

	// Set fullscreen if specified in config
	if g.Config.Render.Window.Fullscreen {
		ebiten.SetFullscreen(true)
		log.Info("Set window to fullscreen mode")
	} else {
		log.WithFields(log.Fields{
			"width":  g.Width,
			"height": g.Height,
		}).Info("Running in windowed mode")
	}

	// Handle graceful shutdown on SIGINT (^C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT)
	log.Debug("Registered signal handlers for graceful shutdown")

	go func() {
		<-quit
		log.Info("Received interrupt signal, shutting down...")
		g.shutdown = true
	}()

	log.Info("Starting game main loop")
	ebiten.RunGame(g)
	log.Info("Game main loop ended")
}

func (g *Game) Update() error {
	// Check for shutdown or escape key
	if g.shutdown {
		log.Debug("Shutdown flag detected, stopping game")
		return StopGame(g)
	}

	if ebiten.IsKeyPressed(ebiten.KeyEscape) {
		log.Debug("Escape key pressed, stopping game")
		return StopGame(g)
	}

	// Update current scene
	activeScene := g.availableScenes[g.activeSceneId]
	nextSceneId := activeScene.Update()

	// Handle scene transitions
	if nextSceneId != g.activeSceneId {
		log.WithFields(log.Fields{
			"from_scene": g.activeSceneId,
			"to_scene":   nextSceneId,
		}).Info("Scene transition detected")

		if nextSceneId == scenes.ExitSceneId {
			log.Info("Exit scene requested, stopping game")
			return StopGame(g)
		}

		// Check if target scene exists
		nextScene, exists := g.availableScenes[nextSceneId]
		if !exists {
			log.WithField("scene_id", nextSceneId).Error("Requested scene does not exist, staying, launch Error Scene")
			// Log g.availableScenes content
			for id := range g.availableScenes {
				log.Debugf("scene_id: %v", id)
			}
			return StopGame(g)
		}

		log.WithField("scene_id", g.activeSceneId).Debug("Calling OnExit for current scene")
		activeScene.OnExit()

		g.activeSceneId = nextSceneId

		if !nextScene.IsLoaded() {
			log.WithField("scene_id", nextSceneId).Debug("Loading scene for first time")
			nextScene.FirstLoad()
		}

		log.WithField("scene_id", nextSceneId).Debug("Calling OnEnter for new scene")
		nextScene.OnEnter()

		log.WithField("active_scene", g.activeSceneId).Info("Scene transition complete")
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw current scene
	activeScene := g.availableScenes[g.activeSceneId]
	activeScene.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	// Log layout changes if they occur
	if outsideWidth != g.Width || outsideHeight != g.Height {
		log.WithFields(log.Fields{
			"outside_width":  outsideWidth,
			"outside_height": outsideHeight,
			"game_width":     g.Width,
			"game_height":    g.Height,
		}).Debug("Layout size mismatch detected")
	}
	return g.Width, g.Height
}
