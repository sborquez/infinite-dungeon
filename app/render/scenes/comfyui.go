package scenes

import (
	"app/services"
	"fmt"
	"image/color"
	"math"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

type ComfyUIScene struct {
	loaded bool
	deps   *Deps

	// Add your scene-specific fields here
	// Text input state
	textInput       string
	cursorVisible   bool
	lastCursorBlink time.Time
	inputActive     bool

	// Animation state for background
	animationTime float64

	// Image generation state
	isGenerating   bool
	generatedImage *ebiten.Image
	currentPrompt  string
	resultChannel  <-chan *services.AsyncImageResult
}

func NewComfyUIScene(deps *Deps) *ComfyUIScene {
	log.Debug("Creating ComfyUI scene")

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
		textInput:       "Enter your prompt here...",
		cursorVisible:   true,
		lastCursorBlink: time.Now(),
		inputActive:     false,
		animationTime:   0.0,
		isGenerating:    false,
		generatedImage:  nil,
		currentPrompt:   "",
		resultChannel:   nil,
	}

	log.WithField("scene_address", &scene).Debug("ComfyUI scene created successfully")
	return scene
}

func (s *ComfyUIScene) GetName() string {
	return "ComfyUI Demo"
}

func (s *ComfyUIScene) Update() SceneId {
	// Update animation time
	s.animationTime += 1.0 / 60.0 // Assuming 60 FPS

	// Check for async image generation results
	s.checkImageGenerationResult()

	// Handle escape key to exit
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		log.Debug("Escape key pressed in ComfyUI scene, exiting")
		return ExitSceneId
	}

	// Handle text input activation
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		if s.inputActive {
			// Submitting text input
			s.inputActive = false
			trimmedText := strings.TrimSpace(s.textInput)

			if trimmedText != "" && trimmedText != "Enter your prompt here..." {
				log.WithField("prompt", trimmedText).Info("Submitting image generation request")
				s.startImageGeneration(trimmedText)
			} else {
				log.Debug("Empty prompt, not generating image")
				// Add placeholder if empty
				if trimmedText == "" {
					s.textInput = "Enter your prompt here..."
				}
			}
		} else {
			// Activating text input
			s.inputActive = true
			log.Debug("Text input activated")
			// Clear placeholder text when activating
			if s.textInput == "Enter your prompt here..." {
				s.textInput = ""
			}
		}
	}

	// Handle text input when active
	if s.inputActive {
		// Handle text input
		inputChars := ebiten.AppendInputChars(nil)
		for _, char := range inputChars {
			if char >= 32 && char <= 126 { // Printable ASCII characters
				s.textInput += string(char)
			}
		}

		// Handle backspace
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) && len(s.textInput) > 0 {
			s.textInput = s.textInput[:len(s.textInput)-1]
		}

		// Handle cursor blinking
		if time.Since(s.lastCursorBlink) > 500*time.Millisecond {
			s.cursorVisible = !s.cursorVisible
			s.lastCursorBlink = time.Now()
		}
	}

	// Handle input and update scene logic
	// Return the SceneId for the next scene or current scene

	return ComfyUISceneId
}

// startImageGeneration initiates async image generation
func (s *ComfyUIScene) startImageGeneration(prompt string) {
	if s.deps == nil || s.deps.ComfyUI == nil {
		log.Error("Cannot start image generation: ComfyUI service not available")
		return
	}

	if !s.deps.ComfyUI.IsRunning() {
		log.Debug("ComfyUI service not running, attempting to start")
		if err := s.deps.ComfyUI.Start(); err != nil {
			log.WithError(err).Error("Failed to start ComfyUI service")
			return
		}
	}

	// Create image request
	imageRequest := services.ImageRequest{
		WorkflowName:  "default_api.json",
		ContentPrompt: prompt,
		Seed:          42,
		Steps:         20,
		Size:          512,
		Ratio:         services.ImageRatioLandscape,
	}

	log.WithFields(log.Fields{
		"workflow":       imageRequest.WorkflowName,
		"content_prompt": imageRequest.ContentPrompt,
		"seed":           imageRequest.Seed,
		"steps":          imageRequest.Steps,
		"ratio":          imageRequest.Ratio,
	}).Info("Starting async image generation")

	// Start async generation
	s.resultChannel = s.deps.ComfyUI.AsyncNewImageFromPrompt(imageRequest)
	s.isGenerating = true
	s.currentPrompt = prompt
}

// checkImageGenerationResult checks for async image generation results
func (s *ComfyUIScene) checkImageGenerationResult() {
	if !s.isGenerating || s.resultChannel == nil {
		return
	}

	// Non-blocking check for results
	select {
	case result := <-s.resultChannel:
		s.handleImageGenerationResult(result)
	default:
		// No result yet, continue loading
	}
}

// handleImageGenerationResult processes the async image generation result
func (s *ComfyUIScene) handleImageGenerationResult(result *services.AsyncImageResult) {
	s.isGenerating = false
	s.resultChannel = nil

	if result.Error != nil {
		log.WithError(result.Error).Error("Image generation failed")
		// Keep placeholder, maybe show error state
		return
	}

	if result.Result != nil && result.Result.Image != nil {
		// Replace old image if exists
		if s.generatedImage != nil {
			s.generatedImage.Deallocate()
		}

		s.generatedImage = result.Result.Image
		bounds := s.generatedImage.Bounds()

		log.WithFields(log.Fields{
			"image_width":  bounds.Dx(),
			"image_height": bounds.Dy(),
			"prompt":       s.currentPrompt,
		}).Info("Image generation completed successfully")
	} else {
		log.Error("Received nil image result")
	}
}

// drawAnimatedBackground renders a cool animated background with gradients and geometric shapes
func (s *ComfyUIScene) drawAnimatedBackground(screen *ebiten.Image) {
	screenWidth := float32(screen.Bounds().Dx())
	screenHeight := float32(screen.Bounds().Dy())

	// Create animated gradient background
	s.drawGradientBackground(screen, screenWidth, screenHeight)

	// Draw particle effect
	s.drawParticleEffect(screen, screenWidth, screenHeight)
}

// drawGradientBackground creates an animated gradient background
func (s *ComfyUIScene) drawGradientBackground(screen *ebiten.Image, width, height float32) {
	// Create animated color values
	time := s.animationTime

	// Base colors that shift over time
	r1 := uint8(20 + 15*math.Sin(time*0.3))
	g1 := uint8(25 + 20*math.Sin(time*0.4+1))
	b1 := uint8(40 + 25*math.Sin(time*0.2+2))

	r2 := uint8(40 + 20*math.Sin(time*0.25+3))
	g2 := uint8(20 + 15*math.Sin(time*0.35+4))
	b2 := uint8(60 + 30*math.Sin(time*0.3+5))

	// Draw gradient rectangles from top to bottom
	steps := 50
	for i := 0; i < steps; i++ {
		ratio := float64(i) / float64(steps)

		// Interpolate colors
		r := uint8(float64(r1)*(1-ratio) + float64(r2)*ratio)
		g := uint8(float64(g1)*(1-ratio) + float64(g2)*ratio)
		b := uint8(float64(b1)*(1-ratio) + float64(b2)*ratio)

		rectHeight := height / float32(steps)
		y := float32(i) * rectHeight

		vector.DrawFilledRect(screen, 0, y, width, rectHeight, color.RGBA{r, g, b, 255}, false)
	}
}

// drawParticleEffect creates a starfield-like particle effect
func (s *ComfyUIScene) drawParticleEffect(screen *ebiten.Image, width, height float32) {
	time := s.animationTime

	// Draw moving particles
	for i := 0; i < 50; i++ {
		// Use deterministic "random" based on index
		seedX := float64(i*123%1000) / 1000.0
		seedY := float64(i*456%1000) / 1000.0
		speedX := float64(i*789%100) / 1000.0
		speedY := float64(i*321%100) / 1000.0

		x := float32(math.Mod(seedX*float64(width)+time*speedX*50, float64(width)))
		y := float32(math.Mod(seedY*float64(height)+time*speedY*30, float64(height)))

		// Particle size varies with time
		size := float32(1 + 2*math.Sin(time*2+float64(i)*0.1))

		alpha := uint8(100 + 100*math.Sin(time*1.5+float64(i)*0.2))
		particleColor := color.RGBA{255, 255, 255, alpha}

		vector.DrawFilledCircle(screen, x, y, size, particleColor, false)
	}
}

// drawGeneratedImage draws a placeholder image in the center with shadow
func (s *ComfyUIScene) drawGeneratedImage(screen *ebiten.Image, screenWidth, screenHeight int) {
	// Calculate image dimensions (16:9 aspect ratio - LANDSCAPE orientation, larger dimension = 512)
	// Since 16:9 means width:height = 16:9, width is larger (landscape)
	baseWidth := float32(512)  // 512 (larger dimension)
	baseHeight := float32(288) // 512 * 9/16

	// Scale relative to screen size (use 80% of screen width as max)
	maxWidth := float32(screenWidth) * 0.8
	scale := maxWidth / baseWidth
	if scale > 1.0 {
		scale = 1.0 // Don't scale up beyond original size
	}

	imageWidth := baseWidth * scale
	imageHeight := baseHeight * scale

	// Center the image
	centerX := float32(screenWidth) / 2
	centerY := float32(screenHeight) / 2
	imageX := centerX - imageWidth/2
	imageY := centerY - imageHeight/2 - (float32(screenHeight) * 0.1)

	// Draw shadow (offset down and right)
	shadowOffset := float32(8 * scale)
	shadowX := imageX + shadowOffset
	shadowY := imageY + shadowOffset
	shadowColor := color.RGBA{0, 0, 0, 100} // Semi-transparent black
	vector.DrawFilledRect(screen, shadowX, shadowY, imageWidth, imageHeight, shadowColor, false)

	if s.generatedImage != nil {
		// Draw the real generated image
		s.drawRealImage(screen, imageX, imageY, imageWidth, imageHeight)
	} else {
		// Draw placeholder background
		placeholderBg := color.RGBA{60, 60, 80, 220}
		vector.DrawFilledRect(screen, imageX, imageY, imageWidth, imageHeight, placeholderBg, false)

		// Draw placeholder border
		borderColor := color.RGBA{100, 100, 120, 255}
		vector.StrokeRect(screen, imageX, imageY, imageWidth, imageHeight, 2, borderColor, false)

		if s.isGenerating {
			// Draw loading animation
			s.drawLoadingAnimation(screen, imageX, imageY, imageWidth, imageHeight)
		} else {
			// Draw placeholder content (grid pattern)
			s.drawPlaceholderContent(screen, imageX, imageY, imageWidth, imageHeight)
		}
	}

	// Draw image info text
	var infoText string
	if s.isGenerating {
		infoText = "Generating image... Please wait"
	} else if s.generatedImage != nil {
		infoText = fmt.Sprintf("Generated: \"%s\"", s.currentPrompt)
		if len(infoText) > 50 {
			infoText = fmt.Sprintf("Generated: \"%.45s...\"", s.currentPrompt)
		}
	} else {
		infoText = "Generated Image Preview (Landscape 16:9)"
	}

	infoY := imageY + imageHeight + 20
	textBg := color.RGBA{0, 0, 0, 150}
	vector.DrawFilledRect(screen, imageX, infoY-5, imageWidth, 25, textBg, false)

	// Center the text
	textOffsetX := (imageWidth - float32(len(infoText)*6)) / 2 // Approximate character width
	ebitenutil.DebugPrintAt(screen, infoText, int(imageX+textOffsetX), int(infoY))
}

// drawRealImage draws the actual generated image scaled to fit the placeholder area
func (s *ComfyUIScene) drawRealImage(screen *ebiten.Image, x, y, width, height float32) {
	if s.generatedImage == nil {
		return
	}

	// Draw the generated image with proper scaling
	options := &ebiten.DrawImageOptions{}

	// Calculate scaling to fit within the placeholder area
	imgBounds := s.generatedImage.Bounds()
	scaleX := width / float32(imgBounds.Dx())
	scaleY := height / float32(imgBounds.Dy())

	// Use the smaller scale to maintain aspect ratio
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}

	// Scale and position the image
	options.GeoM.Scale(float64(scale), float64(scale))

	// Center the scaled image within the placeholder area
	scaledWidth := float32(imgBounds.Dx()) * scale
	scaledHeight := float32(imgBounds.Dy()) * scale
	offsetX := (width - scaledWidth) / 2
	offsetY := (height - scaledHeight) / 2

	options.GeoM.Translate(float64(x+offsetX), float64(y+offsetY))

	screen.DrawImage(s.generatedImage, options)
}

// drawLoadingAnimation draws a spinning loading animation
func (s *ComfyUIScene) drawLoadingAnimation(screen *ebiten.Image, x, y, width, height float32) {
	time := s.animationTime
	centerX := x + width/2
	centerY := y + height/2

	// Draw loading spinner
	numDots := 8
	radius := float32(40)
	dotRadius := float32(6)

	for i := 0; i < numDots; i++ {
		angle := float64(i)*(2*math.Pi/float64(numDots)) + time*3
		dotX := centerX + radius*float32(math.Cos(angle))
		dotY := centerY + radius*float32(math.Sin(angle))

		// Fade dots based on position
		alpha := uint8(100 + 100*math.Sin(time*2+float64(i)*0.5))
		dotColor := color.RGBA{150, 200, 255, alpha}

		vector.DrawFilledCircle(screen, dotX, dotY, dotRadius, dotColor, false)
	}

	// Draw loading text
	loadingText := "Generating..."
	textX := centerX - float32(len(loadingText)*6)/2
	textY := centerY + 60
	ebitenutil.DebugPrintAt(screen, loadingText, int(textX), int(textY))

	// Draw progress dots
	progressDots := int(time*2) % 4
	progressText := "Generating"
	for i := 0; i < progressDots; i++ {
		progressText += "."
	}
	progressX := centerX - float32(len(progressText)*6)/2
	progressY := centerY + 80
	ebitenutil.DebugPrintAt(screen, progressText, int(progressX), int(progressY))
}

// drawPlaceholderContent draws a grid pattern inside the placeholder image
func (s *ComfyUIScene) drawPlaceholderContent(screen *ebiten.Image, x, y, width, height float32) {
	time := s.animationTime

	// Draw a grid pattern
	gridSize := float32(32)
	gridColor := color.RGBA{80, 100, 140, 150}

	// Vertical lines
	for i := gridSize; i < width; i += gridSize {
		vector.StrokeLine(screen, x+i, y, x+i, y+height, 1, gridColor, false)
	}

	// Horizontal lines
	for i := gridSize; i < height; i += gridSize {
		vector.StrokeLine(screen, x, y+i, x+width, y+i, 1, gridColor, false)
	}

	// Draw animated center circle
	centerX := x + width/2
	centerY := y + height/2
	radius := float32(20 + 10*math.Sin(time*2))
	circleColor := color.RGBA{120, 160, 200, 180}
	vector.DrawFilledCircle(screen, centerX, centerY, radius, circleColor, false)

	// Draw corner triangles for visual interest
	triangleSize := float32(20)
	triangleColor := color.RGBA{100, 140, 180, 120}

	// Top-left triangle
	vector.DrawFilledRect(screen, x+5, y+5, triangleSize, triangleSize, triangleColor, false)

	// Top-right triangle
	vector.DrawFilledRect(screen, x+width-triangleSize-5, y+5, triangleSize, triangleSize, triangleColor, false)

	// Bottom-left triangle
	vector.DrawFilledRect(screen, x+5, y+height-triangleSize-5, triangleSize, triangleSize, triangleColor, false)

	// Bottom-right triangle
	vector.DrawFilledRect(screen, x+width-triangleSize-5, y+height-triangleSize-5, triangleSize, triangleSize, triangleColor, false)
}

// drawTextInput renders the text input box and related UI elements at the bottom of the screen
func (s *ComfyUIScene) drawTextInput(screen *ebiten.Image, screenWidth, screenHeight int) {
	// Draw text input box at the bottom
	inputBoxHeight := 40
	inputBoxY := screenHeight - inputBoxHeight - 20
	inputBoxWidth := screenWidth - 40
	inputBoxX := 20

	// Draw input box background
	inputBgColor := color.RGBA{40, 40, 50, 200}
	if s.inputActive {
		inputBgColor = color.RGBA{50, 50, 70, 220}
	}
	vector.DrawFilledRect(screen, float32(inputBoxX), float32(inputBoxY), float32(inputBoxWidth), float32(inputBoxHeight), inputBgColor, false)

	// Draw input box border
	borderColor := color.RGBA{80, 80, 100, 255}
	if s.inputActive {
		borderColor = color.RGBA{100, 150, 200, 255}
	}
	vector.StrokeRect(screen, float32(inputBoxX), float32(inputBoxY), float32(inputBoxWidth), float32(inputBoxHeight), 2, borderColor, false)

	// Display text with cursor if active
	displayText := s.textInput
	if s.inputActive && s.cursorVisible {
		displayText += "|"
	}

	ebitenutil.DebugPrintAt(screen, displayText, inputBoxX+10, inputBoxY+12)

	// Draw status information
	statusY := inputBoxY - 40
	var statusText string

	if s.isGenerating {
		statusText = "Generating image... Please wait"
	} else if s.inputActive {
		statusText = "Type your prompt, press Enter to generate image"
	} else {
		statusText = "Press Enter to activate text input and generate images"
	}

	ebitenutil.DebugPrintAt(screen, statusText, 20, statusY)
}

func (s *ComfyUIScene) Draw(screen *ebiten.Image) {
	// Draw animated background
	s.drawAnimatedBackground(screen)

	screenWidth := screen.Bounds().Dx()
	screenHeight := screen.Bounds().Dy()
	// Draw placeholder image with shadow
	s.drawGeneratedImage(screen, screenWidth, screenHeight)

	// Draw main content text
	ebitenutil.DebugPrintAt(screen, "ComfyUI Image Generation", 20, 20)

	// Draw text input interface
	s.drawTextInput(screen, screenWidth, screenHeight)
}

func (s *ComfyUIScene) FirstLoad() {
	log.Info("Loading ComfyUI scene")

	// Initialize scene resources on first load
	s.loaded = true

	log.Info("ComfyUI scene loaded successfully (background image loading disabled)")
}

func (s *ComfyUIScene) OnEnter() {
	log.Info("Entering ComfyUI scene")

	// Reset text input state
	s.inputActive = false
	s.cursorVisible = true
	s.lastCursorBlink = time.Now()
	s.animationTime = 0.0

	log.Debug("ComfyUI scene entered (background image functionality disabled)")
}

func (s *ComfyUIScene) OnExit() {
	log.Info("Exiting ComfyUI scene")

	// Clean up generated image
	if s.generatedImage != nil {
		log.Debug("Deallocating generated image")
		s.generatedImage.Deallocate()
		s.generatedImage = nil
	}

	// Reset generation state
	s.isGenerating = false
	s.resultChannel = nil
	s.currentPrompt = ""

	log.Debug("ComfyUI scene cleanup completed (background image functionality disabled)")
}

func (s *ComfyUIScene) IsLoaded() bool {
	return s.loaded
}

// Verify interface compliance
var _ Scene = (*ComfyUIScene)(nil)
