package scenes

import "github.com/hajimehoshi/ebiten/v2"

type SceneId uint

const (
	StartSceneId SceneId = iota
	BallsSceneId
	GravitySceneId
	ExitSceneId
)

type Scene interface {
	Update() SceneId
	Draw(screen *ebiten.Image)
	FirstLoad()
	OnEnter()
	OnExit()
	IsLoaded() bool
}
