package main

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// PipelineManager manages the collection of rendering layers and the main render loop.
type PipelineManager struct {
	// layers stores active RenderLayer objects, keyed by their Name.
	layers sync.Map
	// mutex protects access to shared state in the PipelineManager (e.g., isRunning).
	mutex sync.Mutex

	// controller is the interface to send final pixel data to the hardware.
	controller *Controller
	// startTime records when the pipeline began running to calculate elapsed time for Lua scripts.
	startTime time.Time
	// pixelBuffer holds the final RGB data (3 per LED) before transmission.
	pixelBuffer []float64

	// isRunning tracks the state of the render loop.
	isRunning bool
}

// fixColor applies a non-linear brightness correction and color bias to an RGB value.
// It assumes inputs are 0-255 uint8 and returns corrected 0-255 uint8 values.
func fixColor(colorR, colorG, colorB float64) (uint8, uint8, uint8) {
	const MaxValU8 float64 = 255.0
	// The original color fixing logic is preserved, working on 0-255 scale.
	// This helps with the perceived brightness curve and hardware color correction.
	colorROut := math.Pow(colorR/MaxValU8, 2.0) * MaxValU8
	colorGOut := math.Pow(colorG/MaxValU8, 2.0) * (MaxValU8 * (0x88 / MaxValU8))
	colorBOut := math.Pow(colorB/MaxValU8, 2.0) * (MaxValU8 * (0x66 / MaxValU8))
	return uint8(math.Min(255, colorROut)),
		uint8(math.Min(255, colorGOut)),
		uint8(math.Min(255, colorBOut))
}

// NewPipelineManager creates and initializes a new PipelineManager.
func NewPipelineManager(c *Controller) *PipelineManager {
	// Assuming LEDCount is defined elsewhere (e.g., in a main package const)
	p := &PipelineManager{
		controller:  c,
		startTime:   time.Now(),
		pixelBuffer: make([]float64, LEDCount*3),
	}
	// Add a default black base layer.
	p.AddLayer(RenderLayer{
		Name:     "base_black",
		Type:     "BASE",
		// Updated code to use 0.0 float range
		Code:     "for i=0, LEDCount-1 do set_pixel(i, 0.0, 0.0, 0.0) end",
		Priority: 0,
		AddedAt:  p.startTime, // Set AddedAt for the initial layer
	})
	return p
}

// AddLayer adds a new layer to the pipeline or updates an existing one.
func (p *PipelineManager) AddLayer(layer RenderLayer) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	switch layer.Type {
	case "BASE":
		layer.BlendMode = ModeBase
	case "TEMPORARY":
		layer.BlendMode = ModeOverwrite
		layer.AddedAt = time.Now()
	default:
		return fmt.Errorf("未知 Layer Type: %s", layer.Type)
	}

	// Ensure only one BASE layer exists at a time.
	if layer.Type == "BASE" {
		p.layers.Range(func(key, value any) bool {
			l := value.(RenderLayer)
			if l.Type == "BASE" && l.Name != layer.Name {
				p.layers.Delete(key)
			}
			return true
		})
	}

	// If a layer with the same name already exists, update its creation time only if it's TEMPORARY,
	// otherwise just update the layer struct, preserving the original AddedAt for PERSISTENT/BASE layers.
	if existing, ok := p.layers.Load(layer.Name); ok {
		existingLayer := existing.(RenderLayer)
		// Preserve original AddedAt unless it's a new TEMPORARY layer
		if layer.Type != "TEMPORARY" {
			layer.AddedAt = existingLayer.AddedAt
		}
	} else {
		// If it's a completely new layer, set its AddedAt now if not already set (e.g., for BASE/PERSISTENT)
		if layer.AddedAt.IsZero() {
			layer.AddedAt = time.Now()
		}
	}
	
	p.layers.Store(layer.Name, layer)
	fmt.Printf("管线: 添加/更新层 '%s' (%s)\n", layer.Name, layer.Type)
	return nil
}

// RemoveLayer removes a layer from the pipeline by its name.
func (p *PipelineManager) RemoveLayer(name string) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if _, ok := p.layers.Load(name); !ok {
		return fmt.Errorf("层 '%s' 不存在", name)
	}
	p.layers.Delete(name)
	fmt.Printf("管线: 删除层 '%s'\n", name)
	return nil
}

// StartLoop initiates the main rendering loop running at 30 FPS.
func (p *PipelineManager) StartLoop() {
	if p.isRunning {
		return
	}
	p.isRunning = true

	// Ticker runs at 60 frames per second
	ticker := time.NewTicker(time.Second / 60)

	go func() {
		for range ticker.C {
			p.renderFrame()
		}
	}()
}

// renderFrame executes all active layers, composites the result, and sends it to the controller.
func (p *PipelineManager) renderFrame() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	// currentTime is the total elapsed time of the pipeline in seconds
	currentTime := time.Since(p.startTime).Seconds()

	var activeLayers []RenderLayer

	// 1. Check for timeouts and collect active layers
	p.layers.Range(func(key, value any) bool {
		layer := value.(RenderLayer)

		if layer.Type == "TEMPORARY" {
			// Calculate the time elapsed since this specific layer was added
			timeElapsed := currentTime - layer.AddedAt.Sub(p.startTime).Seconds()
			if timeElapsed > layer.TimeoutSeconds && layer.TimeoutSeconds > 0 { // Check if TimeoutSeconds > 0 to prevent accidental removal
				fmt.Printf("管线: 临时层 '%s' 超时，自动删除。\n", layer.Name)
				p.layers.Delete(key)
				return true
			}
		}
		activeLayers = append(activeLayers, layer)
		return true
	})

	// 2. Sort layers by execution order
	sort.Slice(activeLayers, func(i, j int) bool {
		// BASE layer must always be drawn first (lowest priority)
		if activeLayers[i].Type == "BASE" {
			return true
		}
		if activeLayers[j].Type == "BASE" {
			return false
		}

		// For all non-BASE layers, use Priority: lower number means it draws earlier.
		return activeLayers[i].Priority < activeLayers[j].Priority
	})

	// 3. Clear the pixel buffer
	for i := range p.pixelBuffer {
		p.pixelBuffer[i] = 0
	}

	// 4. Execute layers in sorted order
	for _, layer := range activeLayers {
		// Calculate the time elapsed since this specific layer was added
		// This value will be exposed to Lua via get_layer_elapsed_time()
		layerElapsedTime := currentTime - layer.AddedAt.Sub(p.startTime).Seconds()

		// Pass both the total pipeline time (currentTime) and layer-specific elapsed time
		if err := layer.execute(&p.pixelBuffer, currentTime, layerElapsedTime); err != nil {
			// If a layer fails, log the error but continue rendering with other layers.
			fmt.Printf("渲染错误 (%s): %v\n", layer.Name, err)
		}
	}

	var pixelBufferBytes = make([]byte, LEDCount * 3);
	for i := range LEDCount {
		pixelBufferBytes[i * 3], pixelBufferBytes[i * 3 + 1], pixelBufferBytes[i * 3 + 2] = fixColor(
			p.pixelBuffer[i * 3],
			p.pixelBuffer[i * 3 + 1],
			p.pixelBuffer[i * 3 + 2],
		);
	}

	// 5. Send the final composite frame to the hardware
	if err := p.controller.SendColors(pixelBufferBytes); err != nil {
		fmt.Printf("硬件提交错误: %v\n", err)
	}
}
