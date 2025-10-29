package main

import (
	"fmt"
	"math"
	"time"

	lua "github.com/yuin/gopher-lua"
)

// BlendMode represents how a layer's output is combined with the existing pixel buffer.
type BlendMode int

const (
	// ModeOverwrite means the layer completely replaces the existing pixel value.
	ModeOverwrite BlendMode = iota
	// ModeBase is functionally the same as Overwrite but used for the lowest layer.
	ModeBase
)

// setupLuaState initializes a Lua environment with custom global functions.
// It exposes 'get_time', 'get_layer_elapsed_time', 'set_pixel', and 'get_pixel' to the Lua script.
func setupLuaState(L *lua.LState, pixelBuffer *[]float64, pipelineTime, layerElapsedTime float64) {
	L.SetGlobal("LEDCount", lua.LNumber(LEDCount))

	// get_time() returns the current time in seconds since the pipeline started.
	L.SetGlobal("get_time", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(pipelineTime))
		return 1
	}))

	// get_layer_elapsed_time() returns the time elapsed in seconds since this layer was added. (New Function)
	L.SetGlobal("get_layer_elapsed_time", L.NewFunction(func(L *lua.LState) int {
		L.Push(lua.LNumber(layerElapsedTime))
		return 1
	}))

	// get_pixel(index) returns the current R, G, B values of a pixel as 0.0-1.0 floats.
	getPixelFunc := L.NewClosure(func(L *lua.LState) int {
		index := int(L.CheckNumber(1))
		// Safety check for LED index
		if index >= 0 && index < LEDCount {
			buffer := *pixelBuffer
			idx := index * 3

			// Scale 0-255 back to 0.0-1.0 float range for Lua
			L.Push(lua.LNumber(float64(buffer[idx+0]) / 255.0))
			L.Push(lua.LNumber(float64(buffer[idx+1]) / 255.0))
			L.Push(lua.LNumber(float64(buffer[idx+2]) / 255.0))
			return 3
		}
		// Return black (0.0, 0.0, 0.0) for out-of-bounds access
		L.Push(lua.LNumber(0.0))
		L.Push(lua.LNumber(0.0))
		L.Push(lua.LNumber(0.0))
		return 3
	})
	L.SetGlobal("get_pixel", getPixelFunc)

	// set_pixel(index, r, g, b) sets the R, G, B values of a pixel.
	// R, G, B are expected to be 0.0-1.0 floats from the Lua script.
	setPixelFunc := L.NewClosure(func(L *lua.LState) int {
		index := int(L.CheckNumber(1))

		// Check and convert 0.0-1.0 Lua input to 0-255 uint8
		rIn := float64(L.CheckNumber(2))
		gIn := float64(L.CheckNumber(3))
		bIn := float64(L.CheckNumber(4))

		// Scale the input floats (0.0-1.0) to 0-255 uint8 and clamp
		r := math.Max(0, math.Min(255, rIn*255.0))
		g := math.Max(0, math.Min(255, gIn*255.0))
		b := math.Max(0, math.Min(255, bIn*255.0))

		// rFixed, gFixed, bFixed := fixColor(r, g, b)

		if index >= 0 && index < LEDCount {
			buffer := *pixelBuffer
			idx := index * 3

			buffer[idx+0] = r
			buffer[idx+1] = g
			buffer[idx+2] = b
		}
		return 0
	})

	L.SetGlobal("set_pixel", setPixelFunc)
}

// RenderLayer defines a single script layer in the rendering pipeline.
type RenderLayer struct {
	// Name is the unique identifier for the layer.
	Name string `json:"name"`
	// Code is the Lua script to be executed for this layer.
	Code string `json:"code"`
	// Type determines the layer's role ("BASE", "TEMPORARY").
	Type string `json:"type"`
	// Priority dictates the rendering order (lower value draws first).
	Priority int `json:"priority"`
	// BlendMode determines how the layer output is applied to the buffer.
	BlendMode BlendMode `json:"-"`

	// TimeoutSeconds specifies how long a "TEMPORARY" layer should last before removal.
	TimeoutSeconds float64 `json:"timeout"`
	// AddedAt records the time the layer was added for timeout tracking and layer elapsed time calculation.
	AddedAt time.Time `json:"-"`
}

// execute runs the layer's Lua code and applies changes to the pixel buffer.
// It now accepts pipelineTime (total runtime) and layerElapsedTime (layer-specific runtime).
func (l *RenderLayer) execute(pixelBuffer *[]float64, pipelineTime, layerElapsedTime float64) error {
	L := lua.NewState()
	defer L.Close()

	setupLuaState(L, pixelBuffer, pipelineTime, layerElapsedTime)

	if err := L.DoString(l.Code); err != nil {
		return fmt.Errorf("执行 Lua 脚本 '%s' 失败: %w", l.Name, err)
	}
	return nil
}
