package router

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	_const "github.com/luispater/mini-router/const"

	"github.com/gin-gonic/gin"
	"github.com/luispater/mini-router/api"
	"github.com/luispater/mini-router/config"
	"github.com/luispater/mini-router/models"
	"github.com/luispater/mini-router/provider"
)

// / SetupRouter creates and configures the Gin router.
// / cfg: Application configuration.
// / providerRegistry: Provider registry.
// / rateLimitersMutex: Mutex for the rate limiter map.
// / Returns a configured Gin router.
func SetupRouter(cfg *config.Config, providerRegistry map[_const.ProviderType]provider.ProviderFactory) *gin.Engine {
	// Create a new Gin router.
	router := gin.Default()

	// Add middleware.
	// Add CORS middleware.
	router.Use(api.CORSMiddleware())
	// Add logging middleware.
	router.Use(api.LoggingMiddleware())
	// Add error handling middleware.
	router.Use(api.ErrorMiddleware())

	// Health check endpoint.
	// Define the GET request handler for the /health route.
	router.GET("/health", func(c *gin.Context) {
		// Return a JSON response with the status and current time.
		c.JSON(http.StatusOK, gin.H{
			// Status is "ok".
			"status": "ok",
			// Current time, formatted as RFC3339.
			"time": time.Now().Format(time.RFC3339),
		})
	})

	// API v1 route group.
	v1 := router.Group("/v1")
	{
		// Routes that do not require authentication.
		// Define the GET request handler for the /models route.
		v1.GET("/models", func(c *gin.Context) {
			// Get all models from the configuration.
			m := cfg.Models

			// Convert to the response format.
			// Create a slice to store the converted data.
			data := make([]models.DisplayModel, 0)

			names := make(map[string]string)
			// Iterate over all models.
			for _, model := range m {
				// Add the model data to the slice.
				if _, hasKey := names[model.Name]; !hasKey && model.Visible && model.Enabled {
					modality := ""
					inputModalities := make([]string, 0)
					outputModalities := make([]string, 0)
					if model.SupportsCompletion {
						inputModalities = append(inputModalities, "text")
						outputModalities = append(outputModalities, "text")
					} else if model.SupportsImageGen {
						inputModalities = append(inputModalities, "text")
						outputModalities = append(outputModalities, "image")
					}
					if model.SupportsInputImage {
						inputModalities = append(inputModalities, "image")
					}
					modality = fmt.Sprintf("%s->%s", strings.Join(inputModalities, "+"), strings.Join(outputModalities, "+"))

					data = append(data, models.DisplayModel{
						ID:                  model.Name,
						Object:              "model",
						Name:                model.DisplayName,
						Description:         model.Description,
						SupportedParameters: model.SupportedParameters,
						ContextLength:       model.ContextLength,
						MaxCompletionTokens: model.MaxTokens,
						Architecture: struct {
							Modality         string   `json:"modality"`
							InputModalities  []string `json:"input_modalities"`
							OutputModalities []string `json:"output_modalities"`
						}{
							Modality:         modality,
							InputModalities:  inputModalities,
							OutputModalities: outputModalities,
						},
						Pricing: struct {
							Prompt     string `json:"prompt"`
							Completion string `json:"completion"`
						}{
							Prompt:     strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.10f", model.InputPricePerToken), "0"), "."),
							Completion: strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.10f", model.OutputPricePerToken), "0"), "."),
						},
					})
					names[model.Name] = ""
				}
			}

			// Return a JSON response with the model list.
			c.JSON(http.StatusOK, gin.H{
				// The object type is "list".
				"object": "list",
				// Model data.
				"data": data,
			})
		})

		// Route group that requires authentication.
		auth := v1.Group("")
		// Use authentication middleware.
		auth.Use(api.AuthMiddleware(cfg))
		{
			// Chat completion.
			// Define the POST request handler for the /chat/completions route.
			auth.POST("/chat/completions", api.ChatCompletionHandler(cfg, providerRegistry))
		}
	}

	// Return the configured router.
	return router
}
