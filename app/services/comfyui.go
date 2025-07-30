// Package services provides various services for the infinite dungeon application,
// including ComfyUI integration for AI image generation.
package services

import (
	"app/common"
	"bytes"
	"encoding/json"
	"fmt"
	"image/color"
	"image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/hajimehoshi/ebiten/v2"
	log "github.com/sirupsen/logrus"
)

// ComfyUIService provides methods to interact with the ComfyUI WebSocket API.
// It manages connections, workflow execution, and image generation through ComfyUI.
type ComfyUIService struct {
	running  bool           // Current running state of the service
	Config   *common.Config // Application configuration
	BaseURL  string         // Base URL for ComfyUI API
	clientID string         // Unique client identifier for WebSocket connections
}

// ImageRatio represents the aspect ratio options for generated images.
type ImageRatio string

const (
	// ImageRatioSquare represents a 1:1 aspect ratio
	ImageRatioSquare ImageRatio = "SQUARE"
	// ImageRatioLandscape represents a wider than tall aspect ratio
	ImageRatioLandscape ImageRatio = "LANDSCAPE"
	// ImageRatioPortrait represents a taller than wide aspect ratio
	ImageRatioPortrait ImageRatio = "PORTRAIT"
)

// ImageRequest contains all parameters needed to generate an image through ComfyUI.
type ImageRequest struct {
	WorkflowName  string     // Name of the workflow file to use
	ContentPrompt string     // Text prompt describing the desired image content
	Seed          int        // Random seed for reproducible generation
	Steps         int        // Number of diffusion steps for generation
	Size          int        // Base size for image dimensions
	Ratio         ImageRatio // Aspect ratio for the generated image
}

// ImageResult contains the generated image data returned from ComfyUI.
type ImageResult struct {
	Image *ebiten.Image // Generated image ready for use in Ebiten
}

// AsyncImageResult represents the result of an asynchronous image generation request.
// It contains either a successful result or an error, but not both.
type AsyncImageResult struct {
	Result *ImageResult // Generated image result (nil if error occurred)
	Error  error        // Error that occurred during generation (nil if successful)
}

// OUTPUT_NODE_WORKFLOW_TYPE defines the ComfyUI node type used for image output.
const OUTPUT_NODE_WORKFLOW_TYPE = "SaveImageWebsocket"

// WSMessage represents a WebSocket message received from ComfyUI during workflow execution.
type WSMessage struct {
	Type string `json:"type"` // Message type (e.g., "executing")
	Data struct {
		PromptID string  `json:"prompt_id"` // Unique identifier for the prompt
		Node     *string `json:"node"`      // Current executing node ID (nil when done)
	} `json:"data"`
}

// PromptRequest represents the payload sent to ComfyUI to queue a workflow for execution.
type PromptRequest struct {
	Prompt   map[string]interface{} `json:"prompt"`    // The workflow definition
	ClientID string                 `json:"client_id"` // Unique client identifier
}

// QueueResponse represents ComfyUI's response when a workflow is successfully queued.
type QueueResponse struct {
	PromptID string `json:"prompt_id"` // Unique identifier assigned to the queued prompt
}

// NewComfyUIService creates a new ComfyUI WebSocket API service instance.
// It initializes the service with the provided configuration and generates a unique client ID.
func NewComfyUIService(config *common.Config) *ComfyUIService {
	return &ComfyUIService{
		running:  false,
		Config:   config,
		BaseURL:  config.ComfyUI.BaseURL,
		clientID: uuid.New().String(),
	}
}

// Start initializes and starts the ComfyUI service.
// Returns an error if the service fails to start properly.
func (s *ComfyUIService) Start() error {
	log.Info("Starting ComfyUI WebSocket API service")
	s.running = true
	return nil
}

// Stop gracefully shuts down the ComfyUI service.
// This method ensures proper cleanup of resources.
func (s *ComfyUIService) Stop() {
	log.Info("Stopping ComfyUI WebSocket API service")
	s.running = false
}

// IsRunning returns the current running state of the ComfyUI service.
func (s *ComfyUIService) IsRunning() bool {
	return s.running
}

// NewImageFromPrompt generates a new image using the provided ImageRequest parameters.
// This is the main entry point for custom image generation requests.
func (s *ComfyUIService) NewImageFromPrompt(request ImageRequest) (*ImageResult, error) {
	return s.processImageRequest(request)
}

// NewDefaultImageFromPrompt generates an image using predefined default parameters.
// This method is useful for testing or when using standard generation settings.
func (s *ComfyUIService) NewDefaultImageFromPrompt() (*ImageResult, error) {
	return s.processImageRequest(ImageRequest{
		WorkflowName:  "default_api.json",
		ContentPrompt: "A beautiful space station in the sky, seen from the ground",
		Seed:          42,
		Steps:         20,
		Ratio:         ImageRatioPortrait,
		Size:          512,
	})
}

// AsyncNewImageFromPrompt generates an image using the provided ImageRequest parameters.
// This is the main entry point for custom image generation requests. It returns a channel
// that will receive the image result when it is ready. The channel is closed after sending
// the result, so it's safe to use with range loops or single reads.
func (s *ComfyUIService) AsyncNewImageFromPrompt(request ImageRequest) <-chan *AsyncImageResult {
	ch := make(chan *AsyncImageResult, 1) // Buffered to prevent goroutine leak

	go func() {
		defer close(ch) // Always close the channel when done

		image, err := s.processImageRequest(request)
		if err != nil {
			ch <- &AsyncImageResult{
				Result: nil,
				Error:  err,
			}
			return
		}

		ch <- &AsyncImageResult{
			Result: image,
			Error:  nil,
		}
	}()

	return ch
}

// processImageRequest handles the core logic for image generation requests.
// It loads the workflow, updates parameters, establishes WebSocket connection,
// queues the prompt, and retrieves the generated image.
func (s *ComfyUIService) processImageRequest(request ImageRequest) (*ImageResult, error) {
	log.WithFields(log.Fields{
		"workflow_name":  request.WorkflowName,
		"content_prompt": request.ContentPrompt,
		"seed":           request.Seed,
		"steps":          request.Steps,
		"ratio":          request.Ratio,
	}).Info("Processing ComfyUI image request")

	// Load the workflow from the assets folder
	log.WithField("workflow_name", request.WorkflowName).Debug("Loading workflow from assets")
	prompt, err := s.loadPrompt(request.WorkflowName)
	if err != nil {
		log.WithError(err).WithField("workflow_name", request.WorkflowName).Error("Failed to load workflow")
		return nil, err
	}
	prompt = s.updatePrompt(prompt, request)
	log.WithField("prompt_size", len(prompt)).Debug("Updated prompt")

	// Open WebSocket connection
	wsURL := strings.Replace(s.BaseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "ws://", 1)

	// Add /ws path like Python version: ws://127.0.0.1:8000/ws?clientId=...
	if !strings.HasSuffix(wsURL, "/") {
		wsURL += "/"
	}
	wsURL += "ws?clientId=" + s.clientID

	log.WithFields(log.Fields{
		"websocket_url": wsURL,
		"client_id":     s.clientID,
	}).Debug("Attempting WebSocket connection to ComfyUI")

	ws, resp, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.WithError(err).WithFields(log.Fields{
			"websocket_url": wsURL,
			"response":      resp,
		}).Error("Failed to establish WebSocket connection")

		if resp != nil {
			log.WithFields(log.Fields{
				"status_code": resp.StatusCode,
				"status":      resp.Status,
				"headers":     resp.Header,
			}).Error("WebSocket handshake response details")
		}
		return nil, err
	}
	defer ws.Close()
	log.Debug("WebSocket connection established")

	// Send the workflow to the ComfyUI server
	log.WithField("prompt_size", len(prompt)).Debug("Queueing prompt to ComfyUI")
	promptId, err := s.queuePrompt(prompt)
	if err != nil {
		log.WithError(err).Error("Failed to queue prompt to ComfyUI")
		return nil, err
	}
	log.WithField("prompt_id", promptId).Debug("Prompt queued successfully")

	// Get images using the Python function logic
	log.WithField("prompt_id", promptId).Debug("Starting image retrieval from WebSocket")
	outputImages := s.getImages(ws, promptId)

	log.WithFields(log.Fields{
		"output_nodes": len(outputImages),
		"total_images": func() int {
			total := 0
			for _, images := range outputImages {
				total += len(images)
			}
			return total
		}(),
	}).Debug("Retrieved images from ComfyUI")

	// Process the first image if available
	var resultImage *ebiten.Image
	imageProcessed := false

	if len(outputImages) > 0 {
		for nodeId, imageData := range outputImages {
			log.WithFields(log.Fields{
				"node_id":     nodeId,
				"image_count": len(imageData),
			}).Debug("Processing images from node")

			if len(imageData) > 0 {
				log.WithFields(log.Fields{
					"node_id":    nodeId,
					"image_size": len(imageData[0]),
				}).Debug("Decoding PNG image from ComfyUI")

				// Decode PNG data instead of writing raw bytes
				reader := bytes.NewReader(imageData[0])
				decodedImg, err := png.Decode(reader)
				if err != nil {
					log.WithError(err).WithField("node_id", nodeId).Error("Failed to decode PNG image")
					continue
				}

				// Convert to Ebiten image
				bounds := decodedImg.Bounds()
				resultImage = ebiten.NewImageFromImage(decodedImg)

				log.WithFields(log.Fields{
					"image_width":  bounds.Dx(),
					"image_height": bounds.Dy(),
				}).Info("ComfyUI image generated successfully")

				imageProcessed = true
				break
			}
		}
	}

	if !imageProcessed {
		log.Warn("No images were processed from ComfyUI response, using fallback")
		// Create fallback white image
		resultImage = ebiten.NewImage(512, 512)
		resultImage.Fill(color.White)
	}

	return &ImageResult{
		Image: resultImage,
	}, nil
}

// loadPrompt reads and returns the workflow definition from the specified file.
// The workflow file should be located in the configured workflow folder.
func (s *ComfyUIService) loadPrompt(workflowName string) ([]byte, error) {
	workflow, err := os.ReadFile(filepath.Join(s.Config.ComfyUI.WorkflowFolder, workflowName))
	if err != nil {
		return nil, err
	}
	return workflow, nil
}

// getImages listens on the WebSocket connection for workflow execution updates
// and collects generated image data from the output nodes.
// It follows the same logic as the Python implementation for compatibility.
func (s *ComfyUIService) getImages(ws *websocket.Conn, promptId string) map[string][][]byte {
	outputImages := make(map[string][][]byte)
	currentNode := ""

	for {
		messageType, messageData, err := ws.ReadMessage()
		if err != nil {
			log.WithError(err).Error("Failed to read websocket message")
			break
		}

		if messageType == websocket.TextMessage {
			// Handle text message - parse JSON
			var message WSMessage
			if err := json.Unmarshal(messageData, &message); err != nil {
				log.WithError(err).Debug("Failed to parse websocket message")
				continue
			}

			if message.Type == "executing" && message.Data.PromptID == promptId {
				if message.Data.Node == nil {
					// Execution is done
					break
				} else {
					currentNode = *message.Data.Node
				}
			}
		} else if messageType == websocket.BinaryMessage {
			// Handle binary message - collect image data
			// The currentNode will be the node ID (like "11"), not the class type
			if currentNode == "11" { // Output node ID
				// Skip first 8 bytes (header) like in Python version
				if len(messageData) > 8 {
					imageData := messageData[8:]
					imagesOutput := outputImages[currentNode]
					imagesOutput = append(imagesOutput, imageData)
					outputImages[currentNode] = imagesOutput
				}
			}
		}
	}

	return outputImages
}

// findNodeByTitle searches through the workflow nodes to find one with the specified title
// in its _meta field. Returns the node ID and node data if found, empty string and nil otherwise.
func (s *ComfyUIService) findNodeByTitle(promptMap map[string]interface{}, title string) (string, map[string]interface{}) {
	for nodeId, nodeInterface := range promptMap {
		if nodeData, ok := nodeInterface.(map[string]interface{}); ok {
			if meta, ok := nodeData["_meta"].(map[string]interface{}); ok {
				if nodeTitle, ok := meta["title"].(string); ok && nodeTitle == title {
					return nodeId, nodeData
				}
			}
		}
	}
	return "", nil
}

// updateNodeValue updates the 'value' field in the inputs of a node identified by its title.
// This method is used to modify workflow parameters before execution.
// Returns true if the node was found and updated successfully, false otherwise.
func (s *ComfyUIService) updateNodeValue(promptMap map[string]interface{}, title string, value interface{}) bool {
	nodeId, nodeData := s.findNodeByTitle(promptMap, title)
	if nodeData != nil {
		if inputs, ok := nodeData["inputs"].(map[string]interface{}); ok {
			inputs["value"] = value
			log.WithFields(log.Fields{
				"node_id": nodeId,
				"title":   title,
				"value":   value,
			}).Debug("Updated node value")
			return true
		}
	}
	log.WithField("title", title).Warn("Could not find node or inputs for title")
	return false
}

// queuePrompt sends a workflow to ComfyUI for execution via HTTP POST request.
// It parses the workflow, creates the proper request payload, and returns the prompt ID
// that can be used to track execution progress.
func (s *ComfyUIService) queuePrompt(workflow []byte) (string, error) {
	// Parse the workflow JSON
	var prompt map[string]interface{}
	if err := json.Unmarshal(workflow, &prompt); err != nil {
		return "", fmt.Errorf("failed to parse workflow: %w", err)
	}

	// Create the request payload like Python version
	requestPayload := PromptRequest{
		Prompt:   prompt,
		ClientID: s.clientID,
	}

	// Convert to JSON
	jsonData, err := json.Marshal(requestPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// Extract base URL for HTTP endpoint (convert ws:// to http://)
	httpURL := strings.Replace(s.BaseURL, "ws://", "http://", 1)
	httpURL = strings.Replace(httpURL, "/ws", "", 1)
	endpoint := fmt.Sprintf("%s/prompt", httpURL)

	// Make POST request
	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	// Parse response
	var queueResp QueueResponse
	if err := json.NewDecoder(resp.Body).Decode(&queueResp); err != nil {
		return "", fmt.Errorf("failed to parse response: %w", err)
	}

	log.WithFields(log.Fields{
		"prompt_id": queueResp.PromptID,
		"client_id": s.clientID,
	}).Debug("Successfully queued prompt")

	return queueResp.PromptID, nil
}

// updatePrompt modifies a workflow definition with the parameters from an ImageRequest.
// It finds individual nodes by their _meta.title field and updates their input values.
// Only non-zero/non-empty values are applied to avoid overwriting valid defaults.
func (s *ComfyUIService) updatePrompt(prompt []byte, request ImageRequest) []byte {
	var promptMap map[string]interface{}
	if err := json.Unmarshal(prompt, &promptMap); err != nil {
		log.WithError(err).Error("Failed to unmarshal prompt")
		return prompt
	}

	// Update individual nodes by their _meta.title
	if request.Ratio != "" {
		s.updateNodeValue(promptMap, "Ratio", string(request.Ratio))
	}
	if request.ContentPrompt != "" {
		s.updateNodeValue(promptMap, "ContentPrompt", request.ContentPrompt)
	}
	if request.Seed > 0 {
		s.updateNodeValue(promptMap, "Seed", request.Seed)
	}
	if request.Steps > 0 {
		s.updateNodeValue(promptMap, "Steps", request.Steps)
	}
	if request.Size > 0 {
		s.updateNodeValue(promptMap, "Size", float64(request.Size))
	}

	// Debug log the updated request
	log.WithFields(log.Fields{
		"ratio":          request.Ratio,
		"content_prompt": request.ContentPrompt,
		"seed":           request.Seed,
		"steps":          request.Steps,
		"size":           request.Size,
	}).Debug("Updated prompt with request values")

	updatedPrompt, err := json.Marshal(promptMap)
	if err != nil {
		log.WithError(err).Error("Failed to marshal updated prompt")
		return prompt
	}
	return updatedPrompt
}
