package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/luispater/mini-router/config"
	"github.com/luispater/mini-router/provider"
	"github.com/luispater/mini-router/router"
)

const (
	configFile = "config.yaml"
)

// / main function is the entry point of the application.
func main() {
	// gin.SetMode(gin.ReleaseMode)

	// Set the log format, including standard flags and short file names.
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// Print startup information.
	log.Println("Starting AI Router...")

	// Load the configuration file.
	cfg, err := config.LoadConfig(configFile)
	// If loading the configuration fails, log the error and exit.
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Register providers.
	providerRegistry := provider.ProviderRegistry

	// Create a Gin router.
	r := router.SetupRouter(cfg, providerRegistry)

	// Create an HTTP server.
	server := &http.Server{
		// Set the server's listening address.
		Addr: ":" + cfg.Server.Port,
		// Set the server's handler.
		Handler: r,
	}

	// Start the server in a goroutine
	go func() {
		// Print information about the server listening port.
		log.Printf("Server listening on port %s", cfg.Server.Port)
		// Start the server and listen. If an error occurs and it is not a server closed error, log the error and exit.
		if errListenAndServe := server.ListenAndServe(); errListenAndServe != nil && !errors.Is(errListenAndServe, http.ErrServerClosed) {
			log.Fatalf("Failed to start server: %v", errListenAndServe)
		}
	}()

	// Wait for an interrupt signal to gracefully shut down the server.
	quit := make(chan os.Signal, 1)
	// Listen for SIGINT and SIGTERM signals.
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	// Block until a signal is received.
	<-quit
	// Print that the server is shutting down.
	log.Println("Shutting down server...")

	// Create a deadline for the shutdown operation.
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	// Defer canceling the context.
	defer cancel()

	// Shut down the server.
	if err = server.Shutdown(ctx); err != nil {
		// If the server is forced to shut down, log the error and exit.
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	// Print that the server has exited properly.
	log.Println("Server exited properly")
}
