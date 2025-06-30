package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/luispater/mini-router/config"
	"github.com/luispater/mini-router/models"
)

// AuthMiddleware authenticates requests using API keys
func AuthMiddleware(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get the Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Missing API key",
			})
			return
		}

		// Extract the API key
		parts := strings.Split(authHeader, " ")
		var apiKey string
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			apiKey = parts[1]
		} else {
			apiKey = authHeader
		}

		// Find the API key in the in-memory list
		var foundKey *models.APIKey
		for i := range cfg.APIKeys {
			if cfg.APIKeys[i].Key == apiKey {
				foundKey = &cfg.APIKeys[i]
				break
			}
		}

		if foundKey == nil || !foundKey.IsActive {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Invalid API key",
			})
			return
		}

		// Check if the API key has expired
		if !foundKey.ExpiresAt.IsZero() && foundKey.ExpiresAt.Before(time.Now()) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "API key has expired",
			})
			return
		}

		// Store the API key and user in the context
		c.Set("apiKey", *foundKey)

		c.Next()
	}
}

// ErrorMiddleware handles errors and provides consistent error responses
func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there were any errors during the request
		if len(c.Errors) > 0 {
			// Get the last error
			err := c.Errors.Last()

			// Determine the status code
			statusCode := http.StatusInternalServerError
			if c.Writer.Status() != http.StatusOK {
				statusCode = c.Writer.Status()
			}

			// Send a JSON response with the error
			c.JSON(statusCode, gin.H{
				"error": err.Error(),
			})
		}
	}
}

// CORSMiddleware adds CORS headers to responses
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT, DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// LoggingMiddleware logs requests and responses
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Log request details
		statusCode := c.Writer.Status()

		// Log using Gin's logger
		if statusCode >= 400 {
			c.Set("latency", latency)
		} else {
			// Log successful requests
			c.Set("latency", latency)
		}
	}
}
