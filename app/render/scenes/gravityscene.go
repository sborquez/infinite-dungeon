package scenes

import (
	"fmt"
	"image/color"
	"math"
	"math/rand"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type TrailPoint struct {
	X, Y float64
}

type GravityBody struct {
	X, Y    float64
	VX, VY  float64
	Mass    float64
	Color   color.Color
	Trail   []TrailPoint
	Flash   int // frames left to flash after collision
	IsComet bool
}

type GravityScene struct {
	loaded                 bool
	deps                   *Deps
	Bodies                 []GravityBody
	zoom                   float64
	offsetX, offsetY       float64
	lastMouseX, lastMouseY int
	middleDragging         bool
	gravity                float64
	stars                  []TrailPoint
}

const (
	DefaultG         = 1.0
	MinMass          = 10.0
	MaxMass          = 500.0
	TrailLength      = 80  // Increased from 40
	CometTrailLength = 150 // Increased from 80
	NumStars         = 300 // Increased from 120 for better coverage
)

func NewGravityScene(deps *Deps) *GravityScene {
	stars := make([]TrailPoint, NumStars)
	width := float64(deps.Config.Render.Window.Width)
	height := float64(deps.Config.Render.Window.Height)
	for i := range stars {
		stars[i] = TrailPoint{
			X: rand.Float64() * width * 10,  // Increased from 2zoom compatibility
			Y: rand.Float64() * height * 10, // Increased from 2zoom compatibility
		}
	}
	return &GravityScene{
		loaded:  false,
		deps:    deps,
		Bodies:  []GravityBody{},
		zoom:    1.0,
		gravity: DefaultG,
		stars:   stars,
	}
}

func (s *GravityScene) Draw(screen *ebiten.Image) {
	width := float64(s.deps.Config.Render.Window.Width)
	height := float64(s.deps.Config.Render.Window.Height)
	cx, cy := width/2, height/2
	// Draw starfield
	for _, star := range s.stars {
		dx := (star.X+s.offsetX-cx)*s.zoom + cx
		dy := (star.Y+s.offsetY-cy)*s.zoom + cy

		// Only draw stars that are visible on screen
		if dx >= -50 && dx <= width+50 && dy >= -50 && dy <= height+50 {
			// Scale star size with zoom - larger when zoomed out
			starSize := math.Max(1, s.zoom)
			if starSize > 4 {
				starSize = 4
			}

			// Vary star brightness and color
			brightness := uint8(120 + rand.Intn(100))
			col := color.RGBA{brightness, brightness, uint8(brightness + 40), uint8(50 + rand.Intn(100))}

			ebitenutil.DrawRect(screen, dx-starSize/2, dy-starSize/2, starSize, starSize, col)
		}
	}
	// Draw all trails
	for _, b := range s.Bodies {
		trailCol := b.Color
		if b.IsComet {
			trailCol = color.RGBA{255, 255, 255, 180}
		}
		for i := 1; i < len(b.Trail); i++ {
			alpha := uint8(255 * i / len(b.Trail))
			col := trailCol
			if rgba, ok := col.(color.RGBA); ok {
				col = color.RGBA{rgba.R, rgba.G, rgba.B, alpha}
			}
			dx1 := (b.Trail[i-1].X+s.offsetX-cx)*s.zoom + cx
			dy1 := (b.Trail[i-1].Y+s.offsetY-cy)*s.zoom + cy
			dx2 := (b.Trail[i].X+s.offsetX-cx)*s.zoom + cx
			dy2 := (b.Trail[i].Y+s.offsetY-cy)*s.zoom + cy

			// Draw thicker trails for better visibility
			thickness := math.Max(1, s.zoom*0.5)
			if thickness > 3 {
				thickness = 3
			}

			// Draw multiple lines for thickness
			for t := -thickness / 2; t <= thickness/2; t++ {
				ebitenutil.DrawLine(screen, dx1, dy1, dx2+t, dy2+t, col)
			}
		}
	}
	// Draw all bodies
	for _, b := range s.Bodies {
		dx := (b.X+s.offsetX-cx)*s.zoom + cx
		dy := (b.Y+s.offsetY-cy)*s.zoom + cy
		radius := massToRadius(b.Mass) * s.zoom
		col := b.Color
		if b.Flash > 0 {
			col = color.RGBA{255, 255, 0, 255}
		}
		ebitenutil.DrawCircle(screen, dx, dy, radius, col)
		// Draw mass as text
		label := fmt.Sprintf("%.0f", b.Mass)
		ebitenutil.DebugPrintAt(screen, label, int(dx)-8, int(dy)-8)
	}
	// Draw black hole if right mouse held
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		x, y := ebiten.CursorPosition()
		fx, fy := s.screenToWorld(float64(x), float64(y))
		dx := (fx+s.offsetX-cx)*s.zoom + cx
		dy := (fy+s.offsetY-cy)*s.zoom + cy
		ebitenutil.DrawCircle(screen, dx, dy, 24*s.zoom, color.Black)
	}
	// Draw demo name and G
	demo := fmt.Sprintf("Gravity Demo (Q: menu, LMB: random, RMB: float, Hold RMB: remove, Scroll: zoom, MMB: drag, C: comet) | G=%.2f", s.gravity)
	ebitenutil.DebugPrintAt(screen, demo, 40, 40)
	// Draw FPS
	fps := ebiten.ActualFPS()
	fpsStr := fmt.Sprintf("FPS: %0.1f", fps)
	ebitenutil.DebugPrintAt(screen, fpsStr, 10, 10)
}

func (s *GravityScene) FirstLoad()     { s.loaded = true }
func (s *GravityScene) IsLoaded() bool { return s.loaded }
func (s *GravityScene) OnEnter()       {}
func (s *GravityScene) OnExit()        {}

func (s *GravityScene) Update() SceneId {
	if ebiten.IsKeyPressed(ebiten.KeyQ) {
		return StartSceneId
	}
	// --- Variable gravity ---
	_, scrollY := ebiten.Wheel()
	if ebiten.IsKeyPressed(ebiten.KeyShift) && scrollY != 0 {
		factor := 1.0 + 0.2*scrollY
		if s.gravity*factor > 0.01 && s.gravity*factor < 100 {
			s.gravity *= factor
		}
	}
	// --- Zoom ---
	if !ebiten.IsKeyPressed(ebiten.KeyShift) && scrollY != 0 {
		factor := 1.0 + 0.1*scrollY
		if s.zoom*factor > 0.1 && s.zoom*factor < 10 {
			s.zoom *= factor
		}
	}
	// --- Middle mouse drag for panning ---
	mx, my := ebiten.CursorPosition()
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonMiddle) {
		if !s.middleDragging {
			s.middleDragging = true
			s.lastMouseX = mx
			s.lastMouseY = my
		} else {
			dx := mx - s.lastMouseX
			dy := my - s.lastMouseY
			s.offsetX += float64(dx) / s.zoom
			s.offsetY += float64(dy) / s.zoom
			s.lastMouseX = mx
			s.lastMouseY = my
		}
	} else {
		s.middleDragging = false
	}
	// --- Comet spawner ---
	if inpututil.IsKeyJustPressed(ebiten.KeyC) {
		x, y := ebiten.CursorPosition()
		fx, fy := s.screenToWorld(float64(x), float64(y))
		mass := 20.0
		angle := rand.Float64() * 2 * math.Pi
		speed := 16.0 + rand.Float64()*8.0
		vx := math.Cos(angle) * speed
		vy := math.Sin(angle) * speed
		col := color.RGBA{255, 255, 255, 255}
		body := GravityBody{X: fx, Y: fy, VX: vx, VY: vy, Mass: mass, Color: col, IsComet: true}
		s.Bodies = append(s.Bodies, body)
	}
	// --- Spawning on mouse release ---
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		fx, fy := s.screenToWorld(float64(x), float64(y))
		mass := MinMass + rand.Float64()*(MaxMass-MinMass)
		vx := rand.Float64()*4 - 2
		vy := rand.Float64()*4 - 2
		col := massToColor(mass)
		body := GravityBody{X: fx, Y: fy, VX: vx, VY: vy, Mass: mass, Color: col}
		s.Bodies = append(s.Bodies, body)
	}
	if inpututil.IsMouseButtonJustReleased(ebiten.MouseButtonRight) {
		x, y := ebiten.CursorPosition()
		fx, fy := s.screenToWorld(float64(x), float64(y))
		mass := 100.0
		vx, vy := 0.0, 0.0
		col := massToColor(mass)
		body := GravityBody{X: fx, Y: fy, VX: vx, VY: vy, Mass: mass, Color: col}
		if !bodyAt(s.Bodies, fx, fy) {
			s.Bodies = append(s.Bodies, body)
		}
	}
	// Remove bodies under mouse if holding RMB
	if ebiten.IsMouseButtonPressed(ebiten.MouseButtonRight) {
		x, y := ebiten.CursorPosition()
		fx, fy := s.screenToWorld(float64(x), float64(y))
		for i := 0; i < len(s.Bodies); {
			b := s.Bodies[i]
			dist := math.Hypot(b.X-fx, b.Y-fy)
			if dist < massToRadius(b.Mass)+24 {
				s.Bodies = append(s.Bodies[:i], s.Bodies[i+1:]...)
			} else {
				i++
			}
		}
	}
	// --- Gravity ---
	for i := range s.Bodies {
		fx, fy := 0.0, 0.0
		for j := range s.Bodies {
			if i == j {
				continue
			}
			dx := s.Bodies[j].X - s.Bodies[i].X
			dy := s.Bodies[j].Y - s.Bodies[i].Y
			distSq := dx*dx + dy*dy
			if distSq < 1 {
				distSq = 1
			}
			force := s.gravity * s.Bodies[i].Mass * s.Bodies[j].Mass / distSq
			angle := math.Atan2(dy, dx)
			fx += force * math.Cos(angle) / s.Bodies[i].Mass
			fy += force * math.Sin(angle) / s.Bodies[i].Mass
		}
		s.Bodies[i].VX += fx
		s.Bodies[i].VY += fy
	}
	// --- Move bodies (no border, infinite world) and update trails/flash ---
	for i := range s.Bodies {
		b := &s.Bodies[i]
		b.X += b.VX
		b.Y += b.VY
		// Update trail
		trailLen := TrailLength
		if b.IsComet {
			trailLen = CometTrailLength
		}
		b.Trail = append(b.Trail, TrailPoint{b.X, b.Y})
		if len(b.Trail) > trailLen {
			b.Trail = b.Trail[len(b.Trail)-trailLen:]
		}
		if b.Flash > 0 {
			b.Flash--
		}
	}
	// --- Merge on collision ---
	for i := 0; i < len(s.Bodies); i++ {
		for j := i + 1; j < len(s.Bodies); {
			dist := math.Hypot(s.Bodies[i].X-s.Bodies[j].X, s.Bodies[i].Y-s.Bodies[j].Y)
			if dist < massToRadius(s.Bodies[i].Mass)+massToRadius(s.Bodies[j].Mass) {
				// Merge: conserve momentum, sum mass
				totalMass := s.Bodies[i].Mass + s.Bodies[j].Mass
				vx := (s.Bodies[i].VX*s.Bodies[i].Mass + s.Bodies[j].VX*s.Bodies[j].Mass) / totalMass
				vy := (s.Bodies[i].VY*s.Bodies[i].Mass + s.Bodies[j].VY*s.Bodies[j].Mass) / totalMass
				color := massToColor(totalMass)
				s.Bodies[i].X = (s.Bodies[i].X*s.Bodies[i].Mass + s.Bodies[j].X*s.Bodies[j].Mass) / totalMass
				s.Bodies[i].Y = (s.Bodies[i].Y*s.Bodies[i].Mass + s.Bodies[j].Y*s.Bodies[j].Mass) / totalMass
				s.Bodies[i].VX = vx
				s.Bodies[i].VY = vy
				s.Bodies[i].Mass = totalMass
				s.Bodies[i].Color = color
				// Merge trails
				if len(s.Bodies[i].Trail) < len(s.Bodies[j].Trail) {
					s.Bodies[i].Trail = s.Bodies[j].Trail
				}
				s.Bodies[i].Flash = 10
				s.Bodies = append(s.Bodies[:j], s.Bodies[j+1:]...)
			} else {
				j++
			}
		}
	}
	return GravitySceneId
}

// Convert screen (x, y) to world coordinates (taking zoom and pan into account)
func (s *GravityScene) screenToWorld(x, y float64) (float64, float64) {
	width := float64(s.deps.Config.Render.Window.Width)
	height := float64(s.deps.Config.Render.Window.Height)
	cx, cy := width/2, height/2
	fx := (x-cx)/s.zoom + cx - s.offsetX
	fy := (y-cy)/s.zoom + cy - s.offsetY
	return fx, fy
}

func massToRadius(m float64) float64 {
	return 6 + math.Sqrt(m)*1.5 // Not linear, tweak as needed
}

func massToColor(m float64) color.Color {
	// Blue for small, red for big
	t := (m - MinMass) / (MaxMass - MinMass)
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.RGBA{uint8(80 + 175*t), uint8(80 + 40*(1-t)), uint8(255 * (1 - t)), 255}
}

func bodyAt(bodies []GravityBody, x, y float64) bool {
	for _, b := range bodies {
		dist := math.Hypot(b.X-x, b.Y-y)
		if dist < massToRadius(b.Mass) {
			return true
		}
	}
	return false
}

var _ Scene = (*GravityScene)(nil)
