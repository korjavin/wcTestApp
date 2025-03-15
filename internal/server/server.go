package server

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/korjavin/wctestapp/internal/config"
	"github.com/korjavin/wctestapp/internal/relay"
	"github.com/korjavin/wctestapp/internal/wallet"
)

// Server represents the HTTP server
type Server struct {
	config       *config.Config
	httpServer   *http.Server
	relayServer  *relay.RelayServer
	walletClient *wallet.WalletClient
	logger       Logger
}

// Logger interface for logging
type Logger interface {
	Debug(msg string)
	Info(msg string)
	Warn(msg string)
	Error(msg string)
}

// NewServer creates a new server
func NewServer(config *config.Config, logger Logger) *Server {
	// Create the relay server
	relayServer := relay.NewRelayServer(logger)

	// Create the wallet client
	walletClient := wallet.NewWalletClient(config.RelayWebSocketURL(), logger)

	// Create the HTTP server
	httpServer := &http.Server{
		Addr:         config.ServerAddress(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		config:       config,
		httpServer:   httpServer,
		relayServer:  relayServer,
		walletClient: walletClient,
		logger:       logger,
	}
}

// Start starts the server
func (s *Server) Start() error {
	// Create a new router
	router := http.NewServeMux()

	// Set up routes
	s.setupRoutes(router)

	// Set the router as the HTTP handler
	s.httpServer.Handler = router

	// Start the relay server
	s.relayServer.Start()

	// Start the wallet client cleanup task
	s.walletClient.StartCleanupTask()

	// Log the external URL
	s.logger.Info(fmt.Sprintf("External URL: %s", s.config.ExternalURL()))
	s.logger.Info(fmt.Sprintf("Relay WebSocket URL: %s", s.config.RelayWebSocketURL()))

	// Start the HTTP server
	s.logger.Info(fmt.Sprintf("Starting server on %s", s.config.ServerAddress()))
	if s.config.EnableTLS {
		return s.httpServer.ListenAndServeTLS(s.config.CertFile, s.config.KeyFile)
	}
	return s.httpServer.ListenAndServe()
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down server")
	return s.httpServer.Shutdown(ctx)
}

// setupRoutes sets up the HTTP routes
func (s *Server) setupRoutes(router *http.ServeMux) {
	// Static files
	fs := http.FileServer(http.Dir(s.config.StaticDir))
	router.Handle("/static/", http.StripPrefix("/static/", fs))

	// WebSocket relay endpoint
	router.HandleFunc("/relay", s.relayServer.HandleWebSocket)

	// API endpoints
	router.HandleFunc("/api/session/create", s.handleCreateSession)
	router.HandleFunc("/api/session/status", s.handleSessionStatus)
	router.HandleFunc("/api/session/disconnect", s.handleDisconnectSession)
	router.HandleFunc("/api/message/sign", s.handleSignMessage)

	// Web pages
	router.HandleFunc("/", s.handleIndex)
	router.HandleFunc("/connected", s.handleConnected)
}

// GetWalletClient returns the wallet client
func (s *Server) GetWalletClient() *wallet.WalletClient {
	return s.walletClient
}

// GetRelayServer returns the relay server
func (s *Server) GetRelayServer() *relay.RelayServer {
	return s.relayServer
}
