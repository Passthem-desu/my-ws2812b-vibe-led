package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// setupRouter initializes and configures the Gin router with API endpoints for layer management.
func setupRouter(p *PipelineManager) *gin.Engine {
	r := gin.Default()

	api := r.Group("/api/layers")
	{
		// GET /api/layers - Lists all active layers in the pipeline.
		api.GET("/", func(c *gin.Context) {
			layers := make(map[string]RenderLayer)
			// Iterate over the concurrent map to collect all layers
			p.layers.Range(func(key, value any) bool {
				layers[key.(string)] = value.(RenderLayer)
				return true
			})
			c.JSON(http.StatusOK, layers)
		})

		// POST /api/layers - Adds a new layer or updates an existing one.
		api.POST("/", func(c *gin.Context) {
			var layer RenderLayer
			// Bind JSON request body to the RenderLayer struct
			if err := c.ShouldBindJSON(&layer); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
				return
			}

			// Removed Authentication check as requested.

			if err := p.AddLayer(layer); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusCreated, gin.H{"status": "success", "name": layer.Name})
		})

		// DELETE /api/layers/:name - Removes a layer by its unique name.
		api.DELETE("/:name", func(c *gin.Context) {
			name := c.Param("name")

			// Removed Authentication check as requested.

			if err := p.RemoveLayer(name); err != nil {
				c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
				return
			}
			c.JSON(http.StatusOK, gin.H{"status": "deleted", "name": name})
		})
	}

	return r
}
