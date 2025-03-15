package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/korjavin/wctestapp/internal/config"
	"github.com/korjavin/wctestapp/internal/logger"
	"github.com/korjavin/wctestapp/internal/server"
)

func main() {
	// Parse command line flags
	logLevel := flag.String("log-level", "info", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Create logger
	log := logger.NewLogger(logger.LogLevelFromString(*logLevel), "main")
	log.Info("Starting WalletConnect Test App")

	// Load configuration
	cfg := config.LoadFromEnv()
	log.Info(fmt.Sprintf("Server address: %s", cfg.ServerAddress()))
	log.Info(fmt.Sprintf("Relay address: %s", cfg.RelayAddress()))

	// Create server
	srv := server.NewServer(cfg, log)

	// Handle graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in a goroutine
	go func() {
		log.Info("Server starting")
		if err := srv.Start(); err != nil {
			log.Error(fmt.Sprintf("Server error: %v", err))
			stop <- os.Interrupt
		}
	}()

	// Wait for interrupt signal
	<-stop
	log.Info("Shutting down server...")

	// Create a deadline for graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		log.Error(fmt.Sprintf("Server shutdown error: %v", err))
	}

	log.Info("Server stopped")
}
