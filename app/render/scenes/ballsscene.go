// In BallsScene, press Q to return to the StartScene.
package scenes

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	BALL_DENSITY = 0.001 // global density parameter (arbitrary units)
	SPAWN_RATE   = 5     // frames between spawns when holding right mouse
	gridCellSize = 100   // pixels
)

type Ball struct {
	X, Y   float32
	VX, VY float32
	Radius float32
	Color  color.Color
}

type gridCell struct {
	indices []int
}

type BallsScene struct {
	loaded bool

	// For left mouse interaction
	mousePressed   bool
	mousePressTick int

	// For right mouse interaction
	rightMousePressed   bool
	rightMouseSpawnTick int

	// BallsScene
	Balls []Ball
	deps  *Deps
}

func NewBallsScene(deps *Deps) *BallsScene {
	return &BallsScene{
		loaded: false,
		Balls:  []Ball{},
		deps:   deps,
	}
}

func newRandomBall(x, y, radius float32) Ball {
	speed := 1.5 + rand.Float64()*2.5 // 1.5 to 4.0
	vx := float32(speed * float64(rand.Float32()*2-1))
	vy := float32(speed * float64(rand.Float32()*2-1))
	if vx == 0 && vy == 0 {
		vx = 2
		vy = 2
	}
	return Ball{
		X:      x,
		Y:      y,
		VX:     vx,
		VY:     vy,
		Radius: radius,
		Color:  randomColor(),
	}
}

func randomColor() color.Color {
	return color.RGBA{
		R: uint8(rand.Intn(256)),
		G: uint8(rand.Intn(256)),
		B: uint8(rand.Intn(256)),
		A: 255,
	}
}

func (s *BallsScene) Draw(screen *ebiten.Image) {
	// Draw demo name
	ebitenutil.DebugPrintAt(screen, "Balls Physics Demo (press Q to return)", 40, 40)
	// Draw all balls
	for _, b := range s.Balls {
		vector.DrawFilledCircle(screen, b.X, b.Y, b.Radius, b.Color, false)
	}
	// Draw preview ball if mouse is held
	if s.mousePressed {
		size := float32(5 + s.mousePressTick/2)
		if size > 200 {
			size = 200
		}
		x, y := ebiten.CursorPosition()
		vector.DrawFilledCircle(screen, float32(x), float32(y), size, color.RGBA{128, 128, 128, 128}, false)
	}
	// Draw FPS counter
	fps := ebiten.ActualFPS()
	fpsStr := fmt.Sprintf("FPS: %.1f", fps)
	ebitenutil.DebugPrintAt(screen, fpsStr, 10, 10)
}

func (s *BallsScene) FirstLoad() {
	s.loaded = true
}

func (s *BallsScene) IsLoaded() bool {
	return s.loaded
}

func (s *BallsScene) OnEnter() {
}

func (s *BallsScene) OnExit() {
}

func (s *BallsScene) Update() SceneId {
	// Q to return to StartScene
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		return StartSceneId
	}
	// Get actual window size from config
	width := float32(s.deps.Config.Render.Window.Width)
	height := float32(s.deps.Config.Render.Window.Height)
	// Wall and balls collision logic
	s.handleCollisions(width, height)

	// Left mouse interaction (existing)
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
		if !s.mousePressed {
			s.mousePressTick = 0
			s.mousePressed = true
		}
		s.mousePressTick++
	} else if s.mousePressed {
		// On release, add a new ball
		x, y := ebiten.CursorPosition()
		size := float32(5 + s.mousePressTick/2) // Size grows with hold duration
		if size > 200 {
			size = 200
		}
		newBall := newRandomBall(float32(x), float32(y), size)
		s.Balls = append(s.Balls, newBall)
		s.mousePressed = false
	}
	// Right mouse interaction: hold to spawn random balls at SPAWN_RATE
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		if !s.rightMousePressed {
			s.rightMousePressed = true
			s.rightMouseSpawnTick = 0
		}
		s.rightMouseSpawnTick++
		if s.rightMouseSpawnTick%SPAWN_RATE == 0 {
			x, y := ebiten.CursorPosition()
			size := float32(5 + rand.Intn(100))
			newBall := newRandomBall(float32(x), float32(y), size)
			s.Balls = append(s.Balls, newBall)
		}
	} else {
		s.rightMousePressed = false
		s.rightMouseSpawnTick = 0
	}
	return BallsSceneId
}

// --- Scalable collision logic using a uniform grid ---
func (s *BallsScene) handleCollisions(width, height float32) {
	grid := make(map[[2]int]*gridCell)

	// Update ball positions and build grid in a single loop
	for i := range s.Balls {
		b := &s.Balls[i] // Get pointer to the actual ball
		// Update position
		b.X += b.VX
		b.Y += b.VY

		// Check wall collisions
		if b.X < b.Radius {
			b.X = b.Radius
			b.VX = -b.VX
		}
		if b.X > width-b.Radius {
			b.X = width - b.Radius
			b.VX = -b.VX
		}
		if b.Y < b.Radius {
			b.Y = b.Radius
			b.VY = -b.VY
		}
		if b.Y > height-b.Radius {
			b.Y = height - b.Radius
			b.VY = -b.VY
		}

		// Build grid
		minX := int((b.X - b.Radius) / gridCellSize)
		maxX := int((b.X + b.Radius) / gridCellSize)
		minY := int((b.Y - b.Radius) / gridCellSize)
		maxY := int((b.Y + b.Radius) / gridCellSize)
		for gx := minX; gx <= maxX; gx++ {
			for gy := minY; gy <= maxY; gy++ {
				key := [2]int{gx, gy}
				if grid[key] == nil {
					grid[key] = &gridCell{}
				}
				grid[key].indices = append(grid[key].indices, i)
			}
		}
	}
	// Check collisions only within each cell
	checked := make(map[[2]int]struct{})
	for _, cell := range grid {
		indices := cell.indices
		for i := 0; i < len(indices); i++ {
			for j := i + 1; j < len(indices); j++ {
				i1, i2 := indices[i], indices[j]
				pair := [2]int{i1, i2}
				if _, ok := checked[pair]; ok {
					continue
				}
				checked[pair] = struct{}{}
				b1 := &s.Balls[i1]
				b2 := &s.Balls[i2]
				dx := float64(b1.X - b2.X)
				dy := float64(b1.Y - b2.Y)
				dist := math.Hypot(dx, dy)
				minDist := float64(b1.Radius + b2.Radius)
				if dist < minDist && dist > 0 {
					// Calculate masses
					m1 := BALL_DENSITY * math.Pi * math.Pow(float64(b1.Radius), 2)
					m2 := BALL_DENSITY * math.Pi * math.Pow(float64(b2.Radius), 2)
					// Normal vector
					nx := dx / dist
					ny := dy / dist
					// Relative velocity
					dvx := float64(b1.VX - b2.VX)
					dvy := float64(b1.VY - b2.VY)
					// Velocity along the normal
					vn := dvx*nx + dvy*ny
					if vn > 0 {
						continue // balls are moving apart
					}
					// Impulse scalar
					impulse := (2 * vn) / (m1 + m2)
					// Update velocities
					b1.VX = float32(float64(b1.VX) - impulse*m2*nx)
					b1.VY = float32(float64(b1.VY) - impulse*m2*ny)
					b2.VX = float32(float64(b2.VX) + impulse*m1*nx)
					b2.VY = float32(float64(b2.VY) + impulse*m1*ny)
					// Optional: separate balls to prevent sticking
					overlap := minDist - dist
					b1.X += float32(nx * overlap / 2)
					b1.Y += float32(ny * overlap / 2)
					b2.X -= float32(nx * overlap / 2)
					b2.Y -= float32(ny * overlap / 2)
				}
			}
		}
	}
}

var _ Scene = (*BallsScene)(nil)
