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

	sceneMap      map[scenes.SceneId]scenes.Scene
	activeSceneId scenes.SceneId
	shutdown      bool
}

func NewGame(config *common.Config) *Game {
	w := config.Render.Window.Width
	h := config.Render.Window.Height

	// Initialize shared dependencies
	deps := &scenes.Deps{
		Config: config,
	}

	sceneMap := map[scenes.SceneId]scenes.Scene{
		scenes.StartSceneId:   scenes.NewStartScene(deps),
		scenes.BallsSceneId:   scenes.NewBallsScene(deps),
		scenes.GravitySceneId: scenes.NewGravityScene(deps),
	}
	activeSceneId := scenes.StartSceneId
	sceneMap[activeSceneId].FirstLoad()

	return &Game{
		Config:        config,
		sceneMap:      sceneMap,
		activeSceneId: activeSceneId,
		shutdown:      false,

		Width:  w,
		Height: h,
	}
}

func StopGame(g *Game) error {
	log.Info("Shutting down Game.")

	// Stop current scene
	activeScene := g.sceneMap[g.activeSceneId]
	activeScene.OnExit()

	// Exit
	return ebiten.Termination
}

func RunGame(g *Game) {
	ebiten.SetWindowTitle(WINDOW_TITLE)
	// Set fullscreen if specified in config
	if g.Config.Render.Window.Fullscreen {
		ebiten.SetFullscreen(true)
	}

	// Handle graceful shutdown on SIGINT (^C)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGINT)
	go func() {
		<-quit
		log.Info("Received interrupt signal, shutting down...")
		g.shutdown = true
	}()

	ebiten.RunGame(g)
}

func (g *Game) Update() error {
	// Check for shutdown or escape key
	if g.shutdown || ebiten.IsKeyPressed(ebiten.KeyEscape) {
		return StopGame(g)
	}

	// Update current scene
	activeScene := g.sceneMap[g.activeSceneId]
	nextSceneId := activeScene.Update()

	// Handle scene transitions
	if nextSceneId != g.activeSceneId {
		if nextSceneId == scenes.ExitSceneId {
			return StopGame(g)
		}
		g.activeSceneId = nextSceneId
		nextScene := g.sceneMap[nextSceneId]
		if !nextScene.IsLoaded() {
			nextScene.FirstLoad()
		}
		nextScene.OnEnter()
		activeScene.OnExit()
	}

	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	// Draw current scene
	activeScene := g.sceneMap[g.activeSceneId]
	activeScene.Draw(screen)
}

func (g *Game) Layout(outsideWidth, outsideHeight int) (int, int) {
	return g.Width, g.Height
}
