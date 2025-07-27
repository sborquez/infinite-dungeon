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
type ComfyUIService struct {
	running  bool
	Config   *common.Config
	BaseURL  string
	clientID string
}

// string enum for image ratio
type ImageRatio string

const (
	ImageRatioSquare    ImageRatio = "SQUARE"
	ImageRatioLandscape ImageRatio = "LANDSCAPE"
	ImageRatioPortrait  ImageRatio = "PORTRAIT"
)

type ImageRequest struct {
	WorkflowName  string
	ContentPrompt string
	StylePrompt   string
	Seed          int
	Steps         int
	Ratio         ImageRatio
}

type ImageResult struct {
	Image *ebiten.Image
}

const OUTPUT_NODE_WORKFLOW_ID = "11"

type WSMessage struct {
	Type string `json:"type"`
	Data struct {
		PromptID string  `json:"prompt_id"`
		Node     *string `json:"node"`
	} `json:"data"`
}

type PromptRequest struct {
	Prompt   map[string]interface{} `json:"prompt"`
	ClientID string                 `json:"client_id"`
}

type QueueResponse struct {
	PromptID string `json:"prompt_id"`
}

// NewComfyUIService creates a new ComfyUI WebSocket API service.
func NewComfyUIService(config *common.Config) *ComfyUIService {
	return &ComfyUIService{
		running:  false,
		Config:   config,
		BaseURL:  config.ComfyUI.BaseURL,
		clientID: uuid.New().String(),
	}
}

func (s *ComfyUIService) Start() error {
	log.Info("Starting ComfyUI WebSocket API service")
	s.running = true
	return nil
}

func (s *ComfyUIService) Stop() {
	log.Info("Stopping ComfyUI WebSocket API service")
	s.running = false
}

func (s *ComfyUIService) IsRunning() bool {
	return s.running
}

func (s *ComfyUIService) NewImageFromPrompt(request ImageRequest) (*ImageResult, error) {
	return s.processImageRequest(request)
}

func (s *ComfyUIService) NewDefaultImageFromPrompt() (*ImageResult, error) {
	return s.processImageRequest(ImageRequest{
		WorkflowName:  "default.json",
		ContentPrompt: "A beautiful space station in the sky, seen from the ground",
		StylePrompt:   "A beautiful space station in the sky, seen from the ground",
		Seed:          42,
		Steps:         20,
		Ratio:         ImageRatioLandscape,
	})
}

func (s *ComfyUIService) processImageRequest(request ImageRequest) (*ImageResult, error) {
	log.WithFields(log.Fields{
		"workflow_name":  request.WorkflowName,
		"content_prompt": request.ContentPrompt,
		"style_prompt":   request.StylePrompt,
		"seed":           request.Seed,
		"steps":          request.Steps,
		"ratio":          request.Ratio,
	}).Info("Starting ComfyUI image request processing")

	// Load the workflow from the assets folder
	log.WithField("workflow_name", request.WorkflowName).Debug("Loading workflow from assets")
	prompt, err := s.loadPrompt(request.WorkflowName)
	if err != nil {
		log.WithError(err).WithField("workflow_name", request.WorkflowName).Error("Failed to load workflow")
		return nil, err
	}
	log.WithField("prompt_size", len(prompt)).Debug("Successfully loaded workflow")

	// Open WebSocket connection
	wsURL := strings.Replace(s.BaseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "ws://", 1)

	// Add /ws path like Python version: ws://127.0.0.1:8000/ws?clientId=...
	if !strings.HasSuffix(wsURL, "/") {
		wsURL += "/"
	}
	wsURL += "ws?clientId=" + s.clientID

	log.WithFields(log.Fields{
		"original_base_url": s.BaseURL,
		"websocket_url":     wsURL,
		"client_id":         s.clientID,
	}).Info("Attempting WebSocket connection to ComfyUI")

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
	log.Info("Successfully established WebSocket connection to ComfyUI")

	// Send the workflow to the ComfyUI server
	log.WithField("prompt_size", len(prompt)).Debug("Queueing prompt to ComfyUI")
	promptId, err := s.queuePrompt(prompt)
	if err != nil {
		log.WithError(err).Error("Failed to queue prompt to ComfyUI")
		return nil, err
	}
	log.WithField("prompt_id", promptId).Info("Successfully queued prompt to ComfyUI")

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
	}).Info("Retrieved images from ComfyUI")

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
					"node_id":      nodeId,
					"image_width":  bounds.Dx(),
					"image_height": bounds.Dy(),
				}).Info("Successfully decoded and converted ComfyUI image")

				imageProcessed = true
				break
			}
		}
	}

	if !imageProcessed {
		log.Warn("No images were processed from ComfyUI response, creating fallback image")
		// Create fallback white image
		resultImage = ebiten.NewImage(512, 512)
		resultImage.Fill(color.White)
	}

	log.WithField("image_processed", imageProcessed).Info("ComfyUI image request processing completed")
	return &ImageResult{
		Image: resultImage,
	}, nil
}

func (s *ComfyUIService) loadPrompt(workflowName string) ([]byte, error) {
	workflow, err := os.ReadFile(filepath.Join(s.Config.ComfyUI.WorkflowFolder, workflowName))
	if err != nil {
		return nil, err
	}
	return workflow, nil
}

// Implement the Python get_images function logic
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
			if currentNode == OUTPUT_NODE_WORKFLOW_ID {
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
